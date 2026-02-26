//! Governor — token-bucket rate limiter for event emission and API calls.
//!
//! Prevents a runaway process spawner or noisy sensor from flooding NATS or
//! downstream consumers.  Each event type has an independent bucket.

use std::collections::HashMap;
use std::time::Instant;

/// A token-bucket rate limiter.
pub struct Governor {
    buckets: HashMap<String, Bucket>,
    default_rate: u32,
    default_burst: u32,
}

struct Bucket {
    tokens: f64,
    max_tokens: f64,
    refill_rate: f64, // tokens per second
    last_check: Instant,
}

impl Bucket {
    fn new(rate: u32, burst: u32) -> Self {
        Self {
            tokens: burst as f64,
            max_tokens: burst as f64,
            refill_rate: rate as f64,
            last_check: Instant::now(),
        }
    }

    fn try_acquire(&mut self) -> bool {
        let now = Instant::now();
        let elapsed = now.duration_since(self.last_check).as_secs_f64();
        self.last_check = now;

        self.tokens = (self.tokens + elapsed * self.refill_rate).min(self.max_tokens);

        if self.tokens >= 1.0 {
            self.tokens -= 1.0;
            true
        } else {
            false
        }
    }
}

impl Governor {
    /// Create a Governor with default per-second rate and burst capacity.
    /// `rate`: tokens refilled per second per bucket.
    /// `burst`: maximum tokens that can accumulate.
    pub fn new(rate: u32, burst: u32) -> Self {
        Self {
            buckets: HashMap::new(),
            default_rate: rate,
            default_burst: burst,
        }
    }

    /// Try to acquire a token for the named event type.
    /// Returns `true` if allowed, `false` if rate-limited.
    pub fn allow(&mut self, event_type: &str) -> bool {
        let bucket = self
            .buckets
            .entry(event_type.to_string())
            .or_insert_with(|| Bucket::new(self.default_rate, self.default_burst));
        bucket.try_acquire()
    }

    /// Returns the number of active buckets.
    pub fn bucket_count(&self) -> usize {
        self.buckets.len()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn governor_allows_within_burst() {
        let mut gov = Governor::new(10, 5);
        // Should allow up to burst count immediately
        for _ in 0..5 {
            assert!(gov.allow("process"));
        }
        // Next should be denied (no time for refill)
        assert!(!gov.allow("process"));
    }

    #[test]
    fn governor_independent_buckets() {
        let mut gov = Governor::new(10, 2);
        assert!(gov.allow("process"));
        assert!(gov.allow("process"));
        assert!(!gov.allow("process"));
        // Different event type gets its own bucket
        assert!(gov.allow("network"));
        assert!(gov.allow("network"));
        assert!(!gov.allow("network"));
        assert_eq!(gov.bucket_count(), 2);
    }

    #[test]
    fn governor_refills_over_time() {
        let mut gov = Governor::new(100, 1);
        assert!(gov.allow("test"));
        assert!(!gov.allow("test"));
        // Wait for refill
        std::thread::sleep(std::time::Duration::from_millis(20));
        assert!(gov.allow("test"));
    }
}
