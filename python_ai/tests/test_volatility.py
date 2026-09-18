import unittest

from python_ai.app.crypto import _volatility_status


class TestVolatilityStatus(unittest.TestCase):
    def test_volatile_range_is_marked_volatile(self):
        status, pct = _volatility_status(100.0, 110.0, 90.0)
        self.assertEqual(status, "volatile")
        self.assertGreater(pct, 0)

    def test_tight_range_is_marked_stable(self):
        status, pct = _volatility_status(100.0, 101.0, 99.0)
        self.assertEqual(status, "stable")
        self.assertLessEqual(pct, 2.0)

    def test_missing_data_is_unknown(self):
        status, pct = _volatility_status(0.0, 0.0, 0.0)
        self.assertEqual(status, "unknown")
        self.assertEqual(pct, 0.0)


if __name__ == "__main__":
    unittest.main()
