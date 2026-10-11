package com.tencent.qcloud.tuikit.timcommon.util;

/**
 * Serializes Android's external media UI (album, camera, recorder and document picker).
 * A second launch while the first result is pending is rejected instead of losing a result.
 */
public final class MediaLaunchGate {
    // A failed third-party activity can occasionally skip its result callback. Keep the
    // recovery deliberately long so a normal user browsing an album is never interrupted.
    private static final long STALE_LEASE_MS = 10 * 60 * 1000L;
    private static boolean inFlight;
    private static long acquiredAtElapsedMs;

    private MediaLaunchGate() {}

    public static synchronized boolean tryAcquire() {
        long now = android.os.SystemClock.elapsedRealtime();
        if (inFlight && now - acquiredAtElapsedMs >= STALE_LEASE_MS) {
            inFlight = false;
        }
        if (inFlight) {
            return false;
        }
        inFlight = true;
        acquiredAtElapsedMs = now;
        return true;
    }

    public static synchronized void release() {
        inFlight = false;
        acquiredAtElapsedMs = 0L;
    }
}
