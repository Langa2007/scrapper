import unittest

from python_ai.app.crypto import (
    _balance_signal_mix,
    _harmonized_signal,
    _technical_confirmation,
    _volatility_status,
)


class TestVolatilityStatus(unittest.TestCase):
    def test_volatile_range_is_marked_volatile(self):
        status, pct = _volatility_status(100.0, 110.0, 90.0)
        self.assertEqual(status, "volatile")
        self.assertGreater(pct, 0)

    def test_tight_range_is_marked_stable(self):
        status, pct = _volatility_status(100.0, 101.0, 99.0)
        self.assertEqual(status, "stable")
        self.assertLessEqual(pct, 4.0)

    def test_missing_data_is_unknown(self):
        status, pct = _volatility_status(0.0, 0.0, 0.0)
        self.assertEqual(status, "unknown")
        self.assertEqual(pct, 0.0)

    def test_short_mix_keeps_half_volatile_and_half_safer(self):
        signals = [
            {"symbol": "A", "volatility": "volatile", "score": 99},
            {"symbol": "B", "volatility": "volatile", "score": 90},
            {"symbol": "C", "volatility": "volatile", "score": 88},
            {"symbol": "D", "volatility": "moderate", "score": 82},
            {"symbol": "E", "volatility": "stable", "score": 75},
            {"symbol": "F", "volatility": "stable", "score": 70},
        ]
        mixed = _balance_signal_mix(signals, target_count=6)
        self.assertEqual(len(mixed), 6)
        self.assertEqual(sum(1 for s in mixed if s["volatility"] == "volatile"), 3)
        self.assertEqual(sum(1 for s in mixed if s["volatility"] in {"moderate", "stable"}), 3)


class TestTechnicalConfirmation(unittest.TestCase):
    def test_short_requires_reversal_and_volume_confirmation(self):
        closes = [100.0 + index for index in range(20)]
        closes[-1] = closes[-2] - 1.0
        confirmed, score, rsi = _technical_confirmation(closes, [100.0] * 5 + [120.0], "short")
        self.assertTrue(confirmed)
        self.assertGreater(score, 0)
        self.assertGreaterEqual(rsi, 68.0)

    def test_long_requires_reversal_and_volume_confirmation(self):
        closes = [120.0 - index for index in range(20)]
        closes[-1] = closes[-2] + 1.0
        confirmed, _, rsi = _technical_confirmation(closes, [100.0] * 5 + [120.0], "long")
        self.assertTrue(confirmed)
        self.assertLessEqual(rsi, 32.0)

    def test_missing_reversal_is_rejected(self):
        closes = [100.0 + index for index in range(20)]
        confirmed, _, _ = _technical_confirmation(closes, [100.0] * 5 + [120.0], "short")
        self.assertFalse(confirmed)


class TestHarmonizedSignal(unittest.TestCase):
    def test_unconfirmed_analysis_is_hold(self):
        signal, color = _harmonized_signal({"confirmed": False, "score": 110}, "long")
        self.assertEqual((signal, color), ("HOLD", "gray"))

    def test_confirmed_long_and_short_use_trade_direction(self):
        analysis = {"confirmed": True, "score": 95}
        self.assertEqual(_harmonized_signal(analysis, "long"), ("STRONG BUY", "green"))
        self.assertEqual(_harmonized_signal(analysis, "short"), ("STRONG SELL", "red"))

    def test_medium_confirmation_uses_non_strong_label(self):
        signal, color = _harmonized_signal({"confirmed": True, "score": 75}, "short")
        self.assertEqual((signal, color), ("SELL", "orange"))


if __name__ == "__main__":
    unittest.main()
