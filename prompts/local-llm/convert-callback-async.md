# Convert Callback to Async/Await

Convert this callback-style JavaScript/TypeScript code to async/await.

## Rules:
- `callback(err, result)` → `try/catch` with `return result` or `throw err`
- `function(callback)` → `async function()`
- Nested callbacks → sequential awaits
- `Promise` wrapping where needed

## Example Input:
```javascript
function getData(id, callback) {
  fetch('/api/' + id, function(err, response) {
    if (err) {
      callback(err, null);
      return;
    }
    parseJSON(response, function(err, data) {
      if (err) {
        callback(err, null);
        return;
      }
      callback(null, data);
    });
  });
}
```

## Example Output:
```javascript
async function getData(id) {
  const response = await fetch('/api/' + id);
  const data = await parseJSON(response);
  return data;
}
```

---

Output ONLY the converted code. No explanations.

Code to convert:
```javascript
{PASTE CODE HERE}
```
