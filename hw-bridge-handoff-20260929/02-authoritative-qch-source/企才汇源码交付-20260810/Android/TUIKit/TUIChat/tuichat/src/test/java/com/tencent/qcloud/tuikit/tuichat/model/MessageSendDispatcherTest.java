package com.tencent.qcloud.tuikit.tuichat.model;

import static org.junit.Assert.assertEquals;
import static org.junit.Assert.assertFalse;
import static org.junit.Assert.assertTrue;

import com.tencent.imsdk.BaseConstants;
import com.tencent.imsdk.v2.V2TIMMessage;

import org.junit.Test;

import java.util.ArrayList;
import java.util.Comparator;
import java.util.List;

public class MessageSendDispatcherTest {
    @Test
    public void keepsFifoOrderWithinConversation() {
        FakeScheduler scheduler = new FakeScheduler();
        MessageSendDispatcher dispatcher = new MessageSendDispatcher("account", scheduler, () -> 0L);
        List<String> attempts = new ArrayList<>();
        List<MessageSendDispatcher.AttemptCallback> callbacks = new ArrayList<>();

        assertTrue(submit(dispatcher, "c2c:1", "first", attempts, callbacks).isAccepted());
        assertTrue(submit(dispatcher, "c2c:1", "second", attempts, callbacks).isAccepted());
        scheduler.runDue();
        assertEquals(listOf("first"), attempts);

        callbacks.get(0).onSuccess(null);
        scheduler.advanceBy(300L);
        assertEquals(listOf("first", "second"), attempts);
    }

    @Test
    public void keepsMediaAndTextInFifoOrderWithinConversation() {
        FakeScheduler scheduler = new FakeScheduler();
        MessageSendDispatcher dispatcher = new MessageSendDispatcher("account", scheduler, () -> 0L);
        List<String> attempts = new ArrayList<>();
        List<MessageSendDispatcher.AttemptCallback> callbacks = new ArrayList<>();

        assertTrue(dispatcher.submit("c2c:1", MessageSendDispatcher.MessageType.MEDIA, callback -> {
            attempts.add("image");
            callbacks.add(callback);
            return "image-id";
        }, new RecordingListener()).isAccepted());
        assertTrue(dispatcher.submit("c2c:1", MessageSendDispatcher.MessageType.SHORT, callback -> {
            attempts.add("text");
            callbacks.add(callback);
            return "text-id";
        }, new RecordingListener()).isAccepted());

        scheduler.runDue();
        assertEquals(listOf("image"), attempts);
        callbacks.get(0).onSuccess(null);
        scheduler.advanceBy(300L);
        assertEquals(listOf("image", "text"), attempts);
    }

    @Test
    public void retriesOnlyFrequencyErrorsAndIgnoresDuplicateTerminalCallback() {
        FakeScheduler scheduler = new FakeScheduler();
        MessageSendDispatcher dispatcher = new MessageSendDispatcher("account", scheduler, () -> 0L);
        RecordingListener listener = new RecordingListener();
        List<MessageSendDispatcher.AttemptCallback> callbacks = new ArrayList<>();
        int[] attempts = {0};

        MessageSendDispatcher.Submission submission = dispatcher.submit("c2c:1", MessageSendDispatcher.MessageType.SHORT,
            callback -> {
                attempts[0]++;
                callbacks.add(callback);
                return "sdk-" + attempts[0];
            }, listener);
        assertTrue(submission.isAccepted());
        scheduler.runDue();
        callbacks.get(0).onError(BaseConstants.ERR_SDK_COMM_API_CALL_FREQUENCY_LIMIT, "limited");
        assertEquals(0, listener.errorCount);

        scheduler.advanceBy(500L);
        assertEquals(2, attempts[0]);
        callbacks.get(1).onError(6014, "permission denied");
        callbacks.get(1).onSuccess(null);
        assertEquals(1, listener.errorCount);
        assertEquals(0, listener.successCount);
    }

    @Test
    public void rejectsConversationQueueAboveThirtyWithoutDroppingAcceptedTasks() {
        FakeScheduler scheduler = new FakeScheduler();
        MessageSendDispatcher dispatcher = new MessageSendDispatcher("account", scheduler, () -> 0L);
        RecordingListener listener = new RecordingListener();

        for (int i = 0; i < 30; i++) {
            assertTrue(dispatcher.submit("group:1", MessageSendDispatcher.MessageType.SHORT,
                callback -> "sdk-pending", listener).isAccepted());
        }
        assertFalse(dispatcher.submit("group:1", MessageSendDispatcher.MessageType.SHORT,
            callback -> "sdk-rejected", listener).isAccepted());
        assertEquals(30, dispatcher.getPendingCount());
    }

    @Test
    public void allowsAnotherConversationWhileFirstIsInFlight() {
        FakeScheduler scheduler = new FakeScheduler();
        MessageSendDispatcher dispatcher = new MessageSendDispatcher("account", scheduler, () -> 0L);
        List<String> attempts = new ArrayList<>();
        List<MessageSendDispatcher.AttemptCallback> callbacks = new ArrayList<>();

        submit(dispatcher, "c2c:1", "one", attempts, callbacks);
        submit(dispatcher, "c2c:2", "two", attempts, callbacks);
        scheduler.runDue();
        scheduler.advanceBy(300L);
        assertEquals(listOf("one", "two"), attempts);
    }

    @Test
    public void synchronousFrequencyCallbackDoesNotPublishStaleSdkId() {
        FakeScheduler scheduler = new FakeScheduler();
        MessageSendDispatcher dispatcher = new MessageSendDispatcher("account", scheduler, () -> 0L);
        RecordingListener listener = new RecordingListener();
        int[] attempts = {0};

        dispatcher.submit("c2c:1", MessageSendDispatcher.MessageType.SHORT, callback -> {
            attempts[0]++;
            if (attempts[0] == 1) {
                callback.onError(BaseConstants.ERR_SDK_COMM_API_CALL_FREQUENCY_LIMIT, "limited");
            }
            return "sdk-" + attempts[0];
        }, listener);

        scheduler.runDue();
        assertEquals(0, listener.sdkMessageIds.size());
        scheduler.advanceBy(500L);
        assertEquals(listOf("sdk-2"), listener.sdkMessageIds);
    }

    @Test
    public void closeCancelsQueuedWorkAndDetachesInFlightCallback() {
        FakeScheduler scheduler = new FakeScheduler();
        MessageSendDispatcher dispatcher = new MessageSendDispatcher("account", scheduler, () -> 0L);
        RecordingListener listener = new RecordingListener();
        List<MessageSendDispatcher.AttemptCallback> callbacks = new ArrayList<>();
        int[] attempts = {0};

        for (int i = 0; i < 2; i++) {
            dispatcher.submit("c2c:1", MessageSendDispatcher.MessageType.SHORT, callback -> {
                attempts[0]++;
                callbacks.add(callback);
                return "sdk-" + attempts[0];
            }, listener);
        }
        scheduler.runDue();
        dispatcher.close();
        callbacks.get(0).onSuccess(null);
        scheduler.advanceBy(1000L);

        assertEquals(1, attempts[0]);
        assertEquals(0, listener.successCount);
        assertEquals(0, listener.errorCount);
        assertEquals(0, dispatcher.getPendingCount());
    }

    private static MessageSendDispatcher.Submission submit(MessageSendDispatcher dispatcher, String conversation,
                                                            String label, List<String> attempts,
                                                            List<MessageSendDispatcher.AttemptCallback> callbacks) {
        return dispatcher.submit(conversation, MessageSendDispatcher.MessageType.SHORT, callback -> {
            attempts.add(label);
            callbacks.add(callback);
            return "sdk-" + label;
        }, new RecordingListener());
    }

    private static List<String> listOf(String... values) {
        List<String> result = new ArrayList<>();
        java.util.Collections.addAll(result, values);
        return result;
    }

    private static final class RecordingListener implements MessageSendDispatcher.Listener {
        private int successCount;
        private int errorCount;
        private final List<String> sdkMessageIds = new ArrayList<>();

        @Override
        public void onQueued(String clientMessageId) {}

        @Override
        public void onSending(String clientMessageId, int retryCount) {}

        @Override
        public void onProgress(String clientMessageId, int progress) {}

        @Override
        public void onSdkMessageId(String clientMessageId, String sdkMessageId) {
            sdkMessageIds.add(sdkMessageId);
        }

        @Override
        public void onSuccess(String clientMessageId, V2TIMMessage message) {
            successCount++;
        }

        @Override
        public void onError(String clientMessageId, int code, String description) {
            errorCount++;
        }
    }

    private static final class FakeScheduler implements MessageSendDispatcher.Scheduler {
        private final List<ScheduledTask> tasks = new ArrayList<>();
        private long now;

        @Override
        public long now() {
            return now;
        }

        @Override
        public void post(Runnable runnable, long delayMs) {
            tasks.add(new ScheduledTask(now + Math.max(0L, delayMs), runnable));
        }

        private void advanceBy(long millis) {
            now += millis;
            runDue();
        }

        private void runDue() {
            boolean ran;
            do {
                ran = false;
                tasks.sort(Comparator.comparingLong(task -> task.at));
                for (int i = 0; i < tasks.size(); i++) {
                    ScheduledTask task = tasks.get(i);
                    if (task.at <= now) {
                        tasks.remove(i);
                        task.runnable.run();
                        ran = true;
                        break;
                    }
                }
            } while (ran);
        }
    }

    private static final class ScheduledTask {
        private final long at;
        private final Runnable runnable;

        private ScheduledTask(long at, Runnable runnable) {
            this.at = at;
            this.runnable = runnable;
        }
    }
}
