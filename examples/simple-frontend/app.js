const API_BASE = 'http://localhost:8080';
const SERVICE_ID = 'demo-frontend';
const SESSION_ID = crypto.randomUUID();

const messagesEl = document.getElementById('messages');
const userInput = document.getElementById('user-input');
const sendBtn = document.getElementById('send-btn');
const tokenUsage = document.getElementById('token-usage');
const modelInput = document.getElementById('model-input');
const streamToggle = document.getElementById('stream-toggle');

let conversationMessages = [];
let totalTokens = 0;

// --- Init ---

async function init() {
    // Ensure demo-frontend service exists
    try {
        await fetch(`${API_BASE}/services/`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ id: SERVICE_ID, name: 'Demo Frontend' }),
        });
    } catch (e) {
        // Ignore — service may already exist or backend may be down
    }

    await loadHistory();
}

// --- History ---

async function loadHistory() {
    try {
        const res = await fetch(
            `${API_BASE}/services/${SERVICE_ID}/conversations?limit=100&session_id=${SESSION_ID}`
        );
        if (!res.ok) return;

        const data = await res.json();
        if (!data.data || data.data.length === 0) return;

        // API returns newest first — reverse so oldest appears at top
        const conversations = [...data.data].reverse();

        for (const conv of conversations) {
            // Extract last user message from request array
            const requestMsgs = conv.messages?.request || [];
            const lastUserMsg = [...requestMsgs].reverse().find(m => m.role === 'user');
            const userContent = lastUserMsg?.content || '';

            // Extract assistant response — string for streamed, array for non-streamed
            const response = conv.messages?.response;
            let assistantContent = '';
            if (typeof response === 'string') {
                assistantContent = response;
            } else if (Array.isArray(response) && response.length > 0) {
                assistantContent = response[0]?.message?.content || '';
            }

            if (userContent) appendMessage('user', userContent);
            if (assistantContent) appendMessage('assistant', assistantContent);
            conversationMessages.push(
                { role: 'user', content: userContent },
                { role: 'assistant', content: assistantContent }
            );
        }
        scrollToBottom();
    } catch (e) {
        // Backend may not be running yet
    }
}

// --- Send ---

sendBtn.addEventListener('click', send);
userInput.addEventListener('keydown', (e) => {
    if (e.key === 'Enter' && !e.shiftKey) {
        e.preventDefault();
        send();
    }
});

// Auto-resize textarea
userInput.addEventListener('input', () => {
    userInput.style.height = 'auto';
    userInput.style.height = Math.min(userInput.scrollHeight, 120) + 'px';
});

async function send() {
    const text = userInput.value.trim();
    if (!text) return;

    userInput.value = '';
    userInput.style.height = 'auto';
    setLoading(true);

    appendMessage('user', text);
    conversationMessages.push({ role: 'user', content: text });

    try {
        if (streamToggle.checked) {
            await sendMessageStream(text);
        } else {
            await sendMessage(text);
        }
    } catch (err) {
        appendMessage('assistant', `Error: ${err.message}`);
    }

    setLoading(false);
}

async function sendMessage() {
    const res = await fetch(`${API_BASE}/v1/chat/completions`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'X-Service-ID': SERVICE_ID,
            'X-Session-ID': SESSION_ID,
        },
        body: JSON.stringify({
            model: modelInput.value,
            messages: conversationMessages,
        }),
    });

    if (!res.ok) {
        const err = await res.json().catch(() => ({}));
        throw new Error(err.error?.message || `HTTP ${res.status}`);
    }

    const data = await res.json();
    const content = data.choices?.[0]?.message?.content || '';

    appendMessage('assistant', content);
    conversationMessages.push({ role: 'assistant', content });

    if (data.usage) {
        totalTokens += data.usage.total_tokens || 0;
        tokenUsage.textContent = `Tokens: ${totalTokens}`;
    }
}

async function sendMessageStream() {
    const res = await fetch(`${API_BASE}/v1/chat/completions`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'X-Service-ID': SERVICE_ID,
            'X-Session-ID': SESSION_ID,
        },
        body: JSON.stringify({
            model: modelInput.value,
            messages: conversationMessages,
            stream: true,
        }),
    });

    if (!res.ok) {
        const err = await res.json().catch(() => ({}));
        throw new Error(err.error?.message || `HTTP ${res.status}`);
    }

    const bubbleEl = appendMessage('assistant', '');
    let fullContent = '';

    const reader = res.body.getReader();
    const decoder = new TextDecoder();
    let buffer = '';

    while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split('\n');
        buffer = lines.pop(); // keep incomplete line in buffer

        for (const line of lines) {
            const trimmed = line.trim();
            if (!trimmed || !trimmed.startsWith('data: ')) continue;

            const payload = trimmed.slice(6);
            if (payload === '[DONE]') break;

            try {
                const chunk = JSON.parse(payload);
                const delta = chunk.choices?.[0]?.delta?.content;
                if (delta) {
                    fullContent += delta;
                    bubbleEl.textContent = fullContent;
                    scrollToBottom();
                }

                if (chunk.usage) {
                    totalTokens += chunk.usage.total_tokens || 0;
                    tokenUsage.textContent = `Tokens: ${totalTokens}`;
                }
            } catch (e) {
                // skip malformed chunks
            }
        }
    }

    conversationMessages.push({ role: 'assistant', content: fullContent });
}

// --- DOM helpers ---

function appendMessage(role, content) {
    const el = document.createElement('div');
    el.className = `message ${role}`;
    el.textContent = content;
    messagesEl.appendChild(el);
    scrollToBottom();
    return el;
}

function scrollToBottom() {
    messagesEl.scrollTop = messagesEl.scrollHeight;
}

function setLoading(loading) {
    sendBtn.disabled = loading;
    userInput.disabled = loading;
}

// --- Start ---
init();
