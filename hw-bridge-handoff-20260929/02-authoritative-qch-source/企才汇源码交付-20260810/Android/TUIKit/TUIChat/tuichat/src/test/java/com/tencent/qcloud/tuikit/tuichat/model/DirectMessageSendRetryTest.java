package com.tencent.qcloud.tuikit.tuichat.model;

import static org.junit.Assert.assertEquals;

import com.tencent.imsdk.BaseConstants;
import com.tencent.imsdk.v2.V2TIMMessage;

import org.junit.Test;

import java.util.ArrayList;
import java.util.Comparator;
import java.util.List;

public class DirectMessageSendRetryTest {
    @Test
    public void retriesFrequencyErrorAndIgnoresStaleAttemptCallback() {
        FakeScheduler scheduler = new FakeScheduler();
        RecordingListener listener = new RecordingListener();
        List<DirectMessageSendRetry.AttemptCallback> callbacks = new ArrayList<>();
        int[] attempts = {0};
        DirectMessageSendRetry retry = new DirectMessageSendRetry(callback -> {
            attempts[0]++;
            callbacks.add(callback);
            if (attempts[0] == 1) {
                callback.onError(BaseConstants.ERR_SDK_COMM_API_CALL_FREQUENCY_LIMIT, "limited");
            }
            return "sdk-" + attempts[0];
        }, listener, scheduler, () -> 0L);

        assertEquals("sdk-1", retry.start());
        assertEquals(0, listener.errorCount);
        scheduler.advanceBy(500L);
        assertEquals(2, attempts[0]);

        callbacks.get(0).onError(7001, "stale");
        callbacks.get(1).onError(7002, "current");
        assertEquals(1, listener.errorCount);
        assertEquals(7002, listener.lastErrorCode);
    }

    @Test
    public void emptySdkIdFailsExactlyOnce() {
        FakeScheduler scheduler = new FakeScheduler();
        RecordingListener listener = new RecordingListener();
        List<DirectMessageSendRetry.AttemptCallback> callbacks = new ArrayList<>();
        DirectMessageSendRetry retry = new DirectMessageSendRetry(callback -> {
            callbacks.add(callback);
            return "";
        }, listener, scheduler, () -> 0L);

        assertEquals("", retry.start());
        callbacks.get(0).onError(7001, "late");
        assertEquals(1, listener.errorCount);
        assertEquals(-1, listener.lastErrorCode);
    }

    private static final class RecordingListener implements DirectMessageSendRetry.Listener {
        private int errorCount;
        private int lastErrorCode;

        @Override
        public void onProgress(int progress) {}

        @Override
        public void onSuccess(V2TIMMessage message) {}

        @Override
        public void onError(int code, String description) {
            errorCount++;
            lastErrorCode = code;
        }
    }

    private static final class FakeScheduler implements DirectMessageSendRetry.Scheduler {
        private final List<ScheduledTask> tasks = new ArrayList<>();
        private long now;

        @Override
        public void post(Runnable runnable, long delayMs) {
            tasks.add(new ScheduledTask(now + Math.max(0L, delayMs), runnable));
        }

        private void advanceBy(long millis) {
            now += millis;
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
