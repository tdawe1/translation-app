# Guide: Building with the GengoWatcher API

This tutorial will guide you through building a simple Node.js script that interacts with the GengoWatcher API to monitor jobs programmatically.

## Prerequisites
- **Node.js** installed.
- A GengoWatcher **Pro or Enterprise** account.
- Your **API Key** (Generated in User Settings).

---

## 1. Project Initialization

Create a new directory and initialize your project:
```bash
mkdir gengo-integration && cd gengo-integration
npm init -y
npm install axios
```

## 2. Authentication

Create a file named `monitor.js`. We'll start by setting up the API client.

```javascript
const axios = require('axios');

const API_KEY = 'your_api_key_here';
const BASE_URL = 'https://api.gengowatcher.com/api/v1';

const client = axios.create({
  baseURL: BASE_URL,
  headers: {
    'Authorization': `Bearer ${API_KEY}`,
    'Content-Type': 'application/json'
  }
});
```

---

## 3. Configuring the Watcher

Before we start monitoring, let's ensure our filters are set correctly.

```javascript
async function setupWatcher() {
  try {
    const response = await client.put('/watcher/config', {
      min_reward: 20.0,
      included_language_pairs: ['en-ja', 'ja-en']
    });
    console.log('Watcher configured:', response.data.data);
  } catch (error) {
    console.error('Config failed:', error.response.data);
  }
}
```

---

## 4. Starting the Watcher

Now, let's start the background monitoring process.

```javascript
async function startWatcher() {
  try {
    const response = await client.post('/watcher/start');
    console.log(response.data.message);
  } catch (error) {
    console.error('Start failed:', error.response.data);
  }
}
```

---

## 5. Fetching New Jobs

We can poll the API for jobs discovered in the last 10 minutes.

```javascript
async function checkNewJobs() {
  try {
    const response = await client.get('/watcher/jobs?status=new');
    const jobs = response.data.data;
    
    if (jobs.length > 0) {
      console.log(`Found ${jobs.length} new jobs!`);
      jobs.forEach(job => {
        console.log(`- [${job.reward}] ${job.title}: ${job.url}`);
      });
    }
  } catch (error) {
    console.error('Fetch failed:', error.response.data);
  }
}

// Check every 60 seconds
setInterval(checkNewJobs, 60000);
```

---

## 6. Putting it All Together

```javascript
async function main() {
  await setupWatcher();
  await startWatcher();
  console.log('Monitoring started...');
  await checkNewJobs();
}

main();
```

## Summary
You now have a basic integration that:
1. Configures your filters.
2. Starts the watcher.
3. Periodically checks for new matches.

## Next Steps
- [WebSocket API Reference](../api/websocket-api.md) for real-time (non-polling) monitoring.
- [Advanced Filtering Guide](../guides/advanced-filtering.md) to refine your matches.
