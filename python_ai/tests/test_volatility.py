import unittest

from python_ai.app.crypto import _balance_signal_mix, _volatility_status


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


if __name__ == "__main__":
    unittest.main()
