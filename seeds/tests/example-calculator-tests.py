# Example: Test-first implementation
# Claude creates these tests, local LLM writes the implementation

# DO NOT MODIFY THESE TESTS
# Write src/calculator.py to pass them

import pytest
from calculator import Calculator


class TestBasicOperations:
    def test_add_positive_numbers(self):
        calc = Calculator()
        assert calc.add(2, 3) == 5

    def test_add_negative_numbers(self):
        calc = Calculator()
        assert calc.add(-2, -3) == -5

    def test_add_mixed_numbers(self):
        calc = Calculator()
        assert calc.add(-2, 5) == 3

    def test_subtract(self):
        calc = Calculator()
        assert calc.subtract(10, 3) == 7
        assert calc.subtract(3, 10) == -7

    def test_multiply(self):
        calc = Calculator()
        assert calc.multiply(4, 5) == 20
        assert calc.multiply(-2, 3) == -6
        assert calc.multiply(0, 100) == 0

    def test_divide(self):
        calc = Calculator()
        assert calc.divide(10, 2) == 5
        assert calc.divide(7, 2) == 3.5

    def test_divide_by_zero(self):
        calc = Calculator()
        with pytest.raises(ValueError, match="Cannot divide by zero"):
            calc.divide(10, 0)


class TestMemory:
    def test_last_result_stored(self):
        calc = Calculator()
        calc.add(5, 3)
        assert calc.last_result == 8

    def test_last_result_updates(self):
        calc = Calculator()
        calc.add(5, 3)
        calc.multiply(2, 4)
        assert calc.last_result == 8

    def test_clear_resets_memory(self):
        calc = Calculator()
        calc.add(5, 3)
        calc.clear()
        assert calc.last_result == 0


class TestChaining:
    def test_use_last_result(self):
        calc = Calculator()
        calc.add(10, 5)  # 15
        result = calc.add(calc.last_result, 5)  # 15 + 5
        assert result == 20
