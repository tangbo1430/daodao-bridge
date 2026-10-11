package com.tencent.qcloud.tuikit.tuichat.model;

import android.os.Handler;
import android.os.Looper;

import com.tencent.imsdk.v2.V2TIMMessage;

/** Keeps media sends on their direct SDK path while retrying Tencent frequency limits. */
final class DirectMessageSendRetry {
    private static final int MAX_FREQUENCY_RETRIES = 6;
    private static final long MAX_RETRY_DELAY_MS = 8000L;

    interface Operation {
        String send(AttemptCallback callback);
    }

    interface AttemptCallback {
        void onProgress(int progress);

        void onSuccess(V2TIMMessage message);

        void onError(int code, String description);
    }

    interface Listener {
        void onProgress(int progress);

        void onSuccess(V2TIMMessage message);

        void onError(int code, String description);
    }

    interface Scheduler {
        void post(Runnable runnable, long delayMs);
    }

    interface JitterSource {
        long nextJitterMs();
    }

    private final Operation operation;
    private final Listener listener;
    private final Scheduler scheduler;
    private final JitterSource jitterSource;
    private int generation;
    private int retryCount;
    private boolean terminal;

    DirectMessageSendRetry(Operation operation, Listener listener) {
        this(operation, listener, new MainThreadScheduler(), () -> (long) (Math.random() * 251L));
    }

    DirectMessageSendRetry(Operation operation, Listener listener, Scheduler scheduler, JitterSource jitterSource) {
        this.operation = operation;
        this.listener = listener;
        this.scheduler = scheduler;
        this.jitterSource = jitterSource;
    }

    String start() {
        return startAttempt();
    }

    private String startAttempt() {
        final int attemptGeneration;
        synchronized (this) {
            if (terminal) {
                return null;
            }
            attemptGeneration = ++generation;
        }

        String sdkMessageId;
        try {
            sdkMessageId = operation.send(new AttemptCallback() {
                @Override
                public void onProgress(int progress) {
                    synchronized (DirectMessageSendRetry.this) {
                        if (!isCurrentAttempt(attemptGeneration)) {
                            return;
                        }
                    }
                    listener.onProgress(progress);
                }

                @Override
                public void onSuccess(V2TIMMessage message) {
                    finishSuccess(attemptGeneration, message);
                }

                @Override
                public void onError(int code, String description) {
                    finishError(attemptGeneration, code, description);
                }
            });
        } catch (RuntimeException exception) {
            finishError(attemptGeneration, -1,
                exception.getMessage() == null ? "消息发送异常" : exception.getMessage());
            return null;
        }

        if (isEmpty(sdkMessageId)) {
            finishError(attemptGeneration, -1, "消息发送未启动");
        }
        return sdkMessageId;
    }

    private void finishSuccess(int attemptGeneration, V2TIMMessage message) {
        if (message == null) {
            finishError(attemptGeneration, -1, "腾讯IM返回空消息");
            return;
        }
        synchronized (this) {
            if (!isCurrentAttempt(attemptGeneration)) {
                return;
            }
            terminal = true;
            generation++;
        }
        listener.onSuccess(message);
    }

    private void finishError(int attemptGeneration, int code, String description) {
        long retryDelay = -1L;
        synchronized (this) {
            if (!isCurrentAttempt(attemptGeneration)) {
                return;
            }
            if (MessageSendDispatcher.isFrequencyLimitError(code) && retryCount < MAX_FREQUENCY_RETRIES) {
                retryCount++;
                generation++;
                retryDelay = Math.min(500L << Math.min(retryCount - 1, 4), MAX_RETRY_DELAY_MS)
                    + jitterSource.nextJitterMs();
            } else {
                terminal = true;
                generation++;
            }
        }

        if (retryDelay >= 0L) {
            scheduler.post(this::startAttempt, retryDelay);
        } else {
            listener.onError(code, description);
        }
    }

    private boolean isCurrentAttempt(int attemptGeneration) {
        return !terminal && generation == attemptGeneration;
    }

    private static boolean isEmpty(String value) {
        return value == null || value.trim().isEmpty();
    }

    private static final class MainThreadScheduler implements Scheduler {
        private final Handler handler = new Handler(Looper.getMainLooper());

        @Override
        public void post(Runnable runnable, long delayMs) {
            handler.postDelayed(runnable, Math.max(0L, delayMs));
        }
    }
}
