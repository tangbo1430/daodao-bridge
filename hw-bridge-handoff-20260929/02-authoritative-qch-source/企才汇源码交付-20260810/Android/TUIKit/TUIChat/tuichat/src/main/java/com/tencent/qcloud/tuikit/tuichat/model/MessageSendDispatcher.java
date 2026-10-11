package com.tencent.qcloud.tuikit.tuichat.model;

import android.os.Handler;
import android.os.Looper;
import android.text.TextUtils;

import com.tencent.imsdk.BaseConstants;
import com.tencent.imsdk.v2.V2TIMMessage;
import com.tencent.qcloud.tuikit.tuichat.util.TUIChatLog;

import java.util.ArrayDeque;
import java.util.ArrayList;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;
import java.util.UUID;
import java.util.concurrent.atomic.AtomicBoolean;

/**
 * Account-scoped dispatcher for outgoing chat messages. It keeps FIFO order per conversation
 * while allowing two conversations to make progress independently.
 */
public final class MessageSendDispatcher {
    private static final String TAG = "MessageSendDispatcher";
    private static final int MAX_PER_CONVERSATION = 30;
    private static final int MAX_PER_ACCOUNT = 50;
    private static final int MAX_ACTIVE_CONVERSATIONS = 2;
    private static final int MAX_FREQUENCY_RETRIES = 6;
    private static final long GLOBAL_SEND_INTERVAL_MS = 300L;
    private static final long AUDIO_COOLDOWN_MS = 750L;
    private static final long MAX_RETRY_DELAY_MS = 8000L;
    private static final long ATTEMPT_TIMEOUT_MS = 60000L;
    private static final long MEDIA_ATTEMPT_TIMEOUT_MS = 10 * 60 * 1000L;

    public enum MessageType {
        SHORT,
        AUDIO,
        MEDIA
    }

    public interface SendOperation {
        String send(AttemptCallback callback);
    }

    public interface AttemptCallback {
        void onProgress(int progress);

        void onSuccess(V2TIMMessage message);

        void onError(int code, String description);
    }

    public interface Listener {
        void onQueued(String clientMessageId);

        void onSending(String clientMessageId, int retryCount);

        void onProgress(String clientMessageId, int progress);

        default void onSdkMessageId(String clientMessageId, String sdkMessageId) {}

        void onSuccess(String clientMessageId, V2TIMMessage message);

        void onError(String clientMessageId, int code, String description);
    }

    public static final class Submission {
        private final boolean accepted;
        private final String clientMessageId;

        private Submission(boolean accepted, String clientMessageId) {
            this.accepted = accepted;
            this.clientMessageId = clientMessageId;
        }

        public boolean isAccepted() {
            return accepted;
        }

        public String getClientMessageId() {
            return clientMessageId;
        }
    }

    interface Scheduler {
        long now();

        void post(Runnable runnable, long delayMs);
    }

    interface JitterSource {
        long nextJitterMs();
    }

    private static final Map<String, MessageSendDispatcher> ACCOUNT_DISPATCHERS = new LinkedHashMap<>();
    private static String activeAccount = "";

    private final String accountId;
    private final Scheduler scheduler;
    private final JitterSource jitterSource;
    private final Map<String, ConversationQueue> conversationQueues = new LinkedHashMap<>();
    private final List<String> roundRobinKeys = new ArrayList<>();
    private int roundRobinIndex;
    private int totalTaskCount;
    private int activeConversationCount;
    private boolean audioInFlight;
    private long audioCooldownUntil;
    private long nextSendAt;
    private boolean drainScheduled;
    private long scheduledDrainAt = Long.MAX_VALUE;
    private boolean closed;

    private MessageSendDispatcher(String accountId) {
        this(accountId, new MainThreadScheduler(), () -> (long) (Math.random() * 251L));
    }

    MessageSendDispatcher(String accountId, Scheduler scheduler, JitterSource jitterSource) {
        this.accountId = accountId;
        this.scheduler = scheduler;
        this.jitterSource = jitterSource;
    }

    public static synchronized MessageSendDispatcher activateAccount(String accountId) {
        String normalized = accountId == null ? "" : accountId.trim();
        if (normalized.isEmpty()) {
            return null;
        }
        if (!TextUtils.equals(activeAccount, normalized)) {
            MessageSendDispatcher previous = ACCOUNT_DISPATCHERS.remove(activeAccount);
            if (previous != null) {
                previous.close();
            }
            activeAccount = normalized;
        }
        MessageSendDispatcher dispatcher = ACCOUNT_DISPATCHERS.get(normalized);
        if (dispatcher == null || dispatcher.closed) {
            dispatcher = new MessageSendDispatcher(normalized);
            ACCOUNT_DISPATCHERS.put(normalized, dispatcher);
        }
        return dispatcher;
    }

    public static synchronized MessageSendDispatcher getActiveDispatcher() {
        return ACCOUNT_DISPATCHERS.get(activeAccount);
    }

    public static synchronized void deactivateAccount() {
        MessageSendDispatcher dispatcher = ACCOUNT_DISPATCHERS.remove(activeAccount);
        activeAccount = "";
        if (dispatcher != null) {
            dispatcher.close();
        }
    }

    public static synchronized boolean isActiveSdkMessageInFlight(String sdkMessageId) {
        MessageSendDispatcher dispatcher = ACCOUNT_DISPATCHERS.get(activeAccount);
        return dispatcher != null && dispatcher.containsSdkMessageId(sdkMessageId);
    }

    public Submission submit(String conversationId, MessageType type, SendOperation operation, Listener listener) {
        if (TextUtils.isEmpty(conversationId) || type == null || operation == null || listener == null) {
            return new Submission(false, null);
        }

        String clientMessageId = "client-" + UUID.randomUUID();
        synchronized (this) {
            if (closed || totalTaskCount >= MAX_PER_ACCOUNT) {
                return new Submission(false, null);
            }
            ConversationQueue queue = conversationQueues.get(conversationId);
            if (queue != null && queue.tasks.size() >= MAX_PER_CONVERSATION) {
                return new Submission(false, null);
            }
            if (queue == null) {
                queue = new ConversationQueue(conversationId);
                conversationQueues.put(conversationId, queue);
                roundRobinKeys.add(conversationId);
            }
            queue.tasks.offer(new SendTask(clientMessageId, type, operation, listener));
            totalTaskCount++;
        }

        listener.onQueued(clientMessageId);
        TUIChatLog.i(TAG, "accepted account=" + accountSummary(accountId) + " conversation=" + conversationId
            + " clientId=" + clientMessageId + " type=" + type);
        requestDrain(0L);
        return new Submission(true, clientMessageId);
    }

    public synchronized int getPendingCount() {
        return totalTaskCount;
    }

    public void close() {
        List<SendTask> cancelledQueued = new ArrayList<>();
        List<SendTask> detachedInFlight = new ArrayList<>();
        synchronized (this) {
            if (closed) {
                return;
            }
            closed = true;
            for (ConversationQueue queue : conversationQueues.values()) {
                for (SendTask task : queue.tasks) {
                    if (task.inFlight) {
                        detachedInFlight.add(task);
                    } else {
                        cancelledQueued.add(task);
                    }
                }
            }
            conversationQueues.clear();
            roundRobinKeys.clear();
            totalTaskCount = 0;
            activeConversationCount = 0;
            audioInFlight = false;
            drainScheduled = false;
        }
        for (SendTask task : cancelledQueued) {
            task.terminal.compareAndSet(false, true);
        }
        // The SDK call itself is not cancelled. Its late callback is detached from the old account UI.
        for (SendTask task : detachedInFlight) {
            task.terminal.compareAndSet(false, true);
        }
        TUIChatLog.i(TAG, "closed account=" + accountSummary(accountId) + " cancelledQueued=" + cancelledQueued.size()
            + " detachedInFlight=" + detachedInFlight.size());
    }

    private void requestDrain(long delayMs) {
        long targetAt = scheduler.now() + Math.max(0L, delayMs);
        synchronized (this) {
            if (closed || (drainScheduled && targetAt >= scheduledDrainAt)) {
                return;
            }
            drainScheduled = true;
            scheduledDrainAt = targetAt;
        }
        scheduler.post(() -> {
            synchronized (MessageSendDispatcher.this) {
                if (closed || scheduler.now() < scheduledDrainAt) {
                    return;
                }
                drainScheduled = false;
                scheduledDrainAt = Long.MAX_VALUE;
            }
            drain();
        }, Math.max(0L, targetAt - scheduler.now()));
    }

    private void drain() {
        while (true) {
            SendTask task;
            long waitMs;
            synchronized (this) {
                if (closed || activeConversationCount >= MAX_ACTIVE_CONVERSATIONS || totalTaskCount == 0) {
                    return;
                }
                long now = scheduler.now();
                if (now < nextSendAt) {
                    requestDrain(nextSendAt - now);
                    return;
                }
                Selection selection = selectNextTask(now);
                task = selection.task;
                waitMs = selection.waitMs;
                if (task == null) {
                    if (waitMs != Long.MAX_VALUE) {
                        requestDrain(waitMs);
                    }
                    return;
                }
                task.inFlight = true;
                activeConversationCount++;
                if (task.type == MessageType.AUDIO) {
                    audioInFlight = true;
                }
                nextSendAt = now + GLOBAL_SEND_INTERVAL_MS;
            }
            startAttempt(task);
        }
    }

    private Selection selectNextTask(long now) {
        if (roundRobinKeys.isEmpty()) {
            return Selection.none(Long.MAX_VALUE);
        }
        long earliestWait = Long.MAX_VALUE;
        int checked = 0;
        while (checked < roundRobinKeys.size()) {
            if (roundRobinIndex >= roundRobinKeys.size()) {
                roundRobinIndex = 0;
            }
            String key = roundRobinKeys.get(roundRobinIndex);
            roundRobinIndex = (roundRobinIndex + 1) % roundRobinKeys.size();
            checked++;
            ConversationQueue queue = conversationQueues.get(key);
            SendTask task = queue == null ? null : queue.tasks.peek();
            if (task == null || task.inFlight || task.terminal.get()) {
                continue;
            }
            if (task.retryReadyAt > now) {
                earliestWait = Math.min(earliestWait, task.retryReadyAt - now);
                continue;
            }
            if (task.type == MessageType.AUDIO && (audioInFlight || now < audioCooldownUntil)) {
                if (!audioInFlight) {
                    earliestWait = Math.min(earliestWait, audioCooldownUntil - now);
                }
                continue;
            }
            return Selection.of(task);
        }
        return Selection.none(earliestWait);
    }

    private void startAttempt(SendTask task) {
        final int attemptGeneration;
        synchronized (this) {
            attemptGeneration = ++task.attemptGeneration;
        }
        task.listener.onSending(task.clientMessageId, task.retryCount);
        TUIChatLog.i(TAG, "send account=" + accountSummary(accountId) + " clientId=" + task.clientMessageId
            + " type=" + task.type + " retry=" + task.retryCount);
        String sdkMessageId;
        try {
            sdkMessageId = task.operation.send(new AttemptCallback() {
                @Override
                public void onProgress(int progress) {
                    if (isCurrentAttempt(task, attemptGeneration)) {
                        task.listener.onProgress(task.clientMessageId, progress);
                    }
                }

                @Override
                public void onSuccess(V2TIMMessage message) {
                    finishSuccess(task, attemptGeneration, message);
                }

                @Override
                public void onError(int code, String description) {
                    finishError(task, attemptGeneration, code, description);
                }
            });
        } catch (RuntimeException exception) {
            finishError(task, attemptGeneration, -1,
                exception.getMessage() == null ? "消息发送异常" : exception.getMessage());
            return;
        }
        if (!isCurrentAttempt(task, attemptGeneration)) {
            return;
        }
        if (TextUtils.isEmpty(sdkMessageId)) {
            finishError(task, attemptGeneration, -1, "消息发送未启动");
        } else {
            task.sdkMessageId = sdkMessageId;
            task.listener.onSdkMessageId(task.clientMessageId, sdkMessageId);
            long timeoutMs = task.type == MessageType.MEDIA ? MEDIA_ATTEMPT_TIMEOUT_MS : ATTEMPT_TIMEOUT_MS;
            scheduler.post(() -> finishError(task, attemptGeneration, -3, "消息发送超时"), timeoutMs);
        }
    }

    private void finishSuccess(SendTask task, int attemptGeneration, V2TIMMessage message) {
        if (!isCurrentAttempt(task, attemptGeneration)) {
            return;
        }
        if (!task.terminal.compareAndSet(false, true)) {
            return;
        }
        removeCompletedTask(task);
        String sdkId = message == null ? task.sdkMessageId : message.getMsgID();
        TUIChatLog.i(TAG, "success account=" + accountSummary(accountId) + " clientId=" + task.clientMessageId
            + " sdkId=" + safeId(sdkId) + " type=" + task.type + " retries=" + task.retryCount);
        task.listener.onSuccess(task.clientMessageId, message);
        requestDrain(0L);
    }

    private void finishError(SendTask task, int attemptGeneration, int code, String description) {
        if (!isCurrentAttempt(task, attemptGeneration)) {
            return;
        }
        if (isFrequencyLimitError(code) && task.retryCount < MAX_FREQUENCY_RETRIES) {
            long retryDelay;
            synchronized (this) {
                if (closed || task.terminal.get()) {
                    return;
                }
                releaseAttempt(task);
                task.retryCount++;
                retryDelay = Math.min(500L << Math.min(task.retryCount - 1, 4), MAX_RETRY_DELAY_MS)
                    + jitterSource.nextJitterMs();
                task.retryReadyAt = scheduler.now() + retryDelay;
            }
            TUIChatLog.w(TAG, "rate limited account=" + accountSummary(accountId) + " clientId=" + task.clientMessageId
                + " code=" + code + " retry=" + task.retryCount + " delayMs=" + retryDelay);
            requestDrain(retryDelay);
            requestDrain(0L);
            return;
        }
        if (!task.terminal.compareAndSet(false, true)) {
            return;
        }
        removeCompletedTask(task);
        TUIChatLog.e(TAG, "failed account=" + accountSummary(accountId) + " clientId=" + task.clientMessageId
            + " sdkId=" + safeId(task.sdkMessageId) + " type=" + task.type + " retries=" + task.retryCount + " code=" + code);
        task.listener.onError(task.clientMessageId, code, description);
        requestDrain(0L);
    }

    private void removeCompletedTask(SendTask task) {
        synchronized (this) {
            releaseAttempt(task);
            ConversationQueue owner = findOwner(task);
            if (owner != null) {
                owner.tasks.remove(task);
                totalTaskCount = Math.max(0, totalTaskCount - 1);
                if (owner.tasks.isEmpty()) {
                    conversationQueues.remove(owner.conversationId);
                    int removedIndex = roundRobinKeys.indexOf(owner.conversationId);
                    if (removedIndex >= 0) {
                        roundRobinKeys.remove(removedIndex);
                        if (removedIndex < roundRobinIndex) {
                            roundRobinIndex--;
                        }
                    }
                }
            }
        }
    }

    private void releaseAttempt(SendTask task) {
        if (!task.inFlight) {
            return;
        }
        task.inFlight = false;
        activeConversationCount = Math.max(0, activeConversationCount - 1);
        if (task.type == MessageType.AUDIO) {
            audioInFlight = false;
            audioCooldownUntil = Math.max(audioCooldownUntil, scheduler.now() + AUDIO_COOLDOWN_MS);
        }
    }

    private synchronized boolean isCurrentAttempt(SendTask task, int attemptGeneration) {
        return !closed && !task.terminal.get() && task.inFlight && task.attemptGeneration == attemptGeneration;
    }

    private synchronized boolean containsSdkMessageId(String sdkMessageId) {
        if (TextUtils.isEmpty(sdkMessageId)) {
            return false;
        }
        for (ConversationQueue queue : conversationQueues.values()) {
            for (SendTask task : queue.tasks) {
                if (!task.terminal.get() && TextUtils.equals(sdkMessageId, task.sdkMessageId)) {
                    return true;
                }
            }
        }
        return false;
    }

    private ConversationQueue findOwner(SendTask task) {
        for (ConversationQueue queue : conversationQueues.values()) {
            if (queue.tasks.contains(task)) {
                return queue;
            }
        }
        return null;
    }

    static boolean isFrequencyLimitError(int code) {
        return code == BaseConstants.ERR_SDK_COMM_API_CALL_FREQUENCY_LIMIT
            || code == BaseConstants.ERR_SDK_NET_FREQ_LIMIT
            || code == BaseConstants.ERR_SDK_NET_REQ_COUNT_LIMIT
            || code == BaseConstants.ERR_SVR_SSO_FREQ_LIMIT
            || code == BaseConstants.ERR_SVR_SSO_FREQUENCY_LIMIT
            || code == BaseConstants.ERR_SVR_COMM_REST_FREQ_LIMIT
            || code == BaseConstants.ERR_SVR_COMM_SDKAPPID_FREQ_LIMIT
            || code == BaseConstants.ERR_SVR_COMM_REQ_FREQ_LIMIT
            || code == BaseConstants.ERR_SVR_COMM_REQ_FREQ_LIMIT_EX
            || code == BaseConstants.ERR_SVR_GROUP_FREQ_LIMIT
            || code == BaseConstants.ERR_SVR_GROUP_SEND_MSG_FREQ_LIMIT;
    }

    private static String accountSummary(String account) {
        if (TextUtils.isEmpty(account)) {
            return "empty";
        }
        return Integer.toHexString(account.hashCode());
    }

    private static String safeId(String id) {
        return TextUtils.isEmpty(id) ? "empty" : id;
    }

    private static final class ConversationQueue {
        private final String conversationId;
        private final ArrayDeque<SendTask> tasks = new ArrayDeque<>();

        private ConversationQueue(String conversationId) {
            this.conversationId = conversationId;
        }
    }

    private static final class SendTask {
        private final String clientMessageId;
        private final MessageType type;
        private final SendOperation operation;
        private final Listener listener;
        private final AtomicBoolean terminal = new AtomicBoolean(false);
        private int retryCount;
        private int attemptGeneration;
        private long retryReadyAt;
        private boolean inFlight;
        private String sdkMessageId;

        private SendTask(String clientMessageId, MessageType type, SendOperation operation, Listener listener) {
            this.clientMessageId = clientMessageId;
            this.type = type;
            this.operation = operation;
            this.listener = listener;
        }
    }

    private static final class Selection {
        private final SendTask task;
        private final long waitMs;

        private Selection(SendTask task, long waitMs) {
            this.task = task;
            this.waitMs = waitMs;
        }

        private static Selection of(SendTask task) {
            return new Selection(task, 0L);
        }

        private static Selection none(long waitMs) {
            return new Selection(null, waitMs);
        }
    }

    private static final class MainThreadScheduler implements Scheduler {
        private final Handler handler = new Handler(Looper.getMainLooper());

        @Override
        public long now() {
            return android.os.SystemClock.uptimeMillis();
        }

        @Override
        public void post(Runnable runnable, long delayMs) {
            handler.postDelayed(runnable, Math.max(0L, delayMs));
        }
    }
}
