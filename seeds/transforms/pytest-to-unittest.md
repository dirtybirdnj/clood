# Transform: pytest to unittest

Convert pytest-style tests to unittest.

## Transformation Rules:

| pytest | unittest |
|--------|----------|
| `assert x == y` | `self.assertEqual(x, y)` |
| `assert x != y` | `self.assertNotEqual(x, y)` |
| `assert x` | `self.assertTrue(x)` |
| `assert not x` | `self.assertFalse(x)` |
| `assert x is None` | `self.assertIsNone(x)` |
| `assert x is not None` | `self.assertIsNotNone(x)` |
| `assert x in y` | `self.assertIn(x, y)` |
| `assert x not in y` | `self.assertNotIn(x, y)` |
| `assert isinstance(x, T)` | `self.assertIsInstance(x, T)` |
| `with pytest.raises(E):` | `with self.assertRaises(E):` |
| `@pytest.fixture` | `setUp(self)` method |
| `def test_foo():` | `def test_foo(self):` |

## Structure Changes:
- Add `import unittest`
- Remove `import pytest`
- Wrap tests in `class TestX(unittest.TestCase):`
- Add `self` parameter to all test methods
- Convert fixtures to `setUp(self)` and `tearDown(self)`

## Example Input:
```python
import pytest

@pytest.fixture
def sample_list():
    return [1, 2, 3]

def test_list_length(sample_list):
    assert len(sample_list) == 3

def test_list_contains(sample_list):
    assert 2 in sample_list
    assert 5 not in sample_list

def test_division_by_zero():
    with pytest.raises(ZeroDivisionError):
        1 / 0
```

## Example Output:
```python
import unittest

class TestList(unittest.TestCase):
    def setUp(self):
        self.sample_list = [1, 2, 3]

    def test_list_length(self):
        self.assertEqual(len(self.sample_list), 3)

    def test_list_contains(self):
        self.assertIn(2, self.sample_list)
        self.assertNotIn(5, self.sample_list)

    def test_division_by_zero(self):
        with self.assertRaises(ZeroDivisionError):
            1 / 0


if __name__ == '__main__':
    unittest.main()
```

---

Output ONLY the converted code. No explanations.

Code to convert:
```python
{PASTE CODE HERE}
```
