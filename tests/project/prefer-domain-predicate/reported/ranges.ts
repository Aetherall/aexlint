if (response.status >= 200 && response.status < 300) use(response);
if (200 <= response.status && 300 > response.status) use(response);
if (value >= -5 && value <= +5) use(value);
if (value >= -5n && value <= 5n) use(value);
