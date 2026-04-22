const apiBaseUrl = window.location.protocol === "file:" ? "http://localhost:8080" : "";

export async function streamTravelAdvice(payload, handlers, signal) {
    const response = await fetch(`${apiBaseUrl}/api/travel-advisor/stream`, {
        method: "POST",
        headers: {
            "Content-Type": "application/json"
        },
        body: JSON.stringify(payload),
        signal
    });

    if (!response.ok || !response.body) {
        throw new Error(`请求失败：${response.status}`);
    }

    const reader = response.body.getReader();
    const decoder = new TextDecoder("utf-8");
    let buffer = "";

    while (true) {
        const { done, value } = await reader.read();
        if (done) {
            break;
        }

        buffer += decoder.decode(value, { stream: true });
        const parts = buffer.split("\n\n");
        buffer = parts.pop() || "";

        parts.forEach((part) => {
            const event = parseSseEvent(part);
            if (!event) {
                return;
            }

            if (event.type === "context") {
                handlers.onContext?.(event.data);
            }

            if (event.type === "delta") {
                handlers.onDelta?.(event.data.content || "");
            }

            if (event.type === "error") {
                handlers.onError?.(event.data.message || "流式响应失败");
            }

            if (event.type === "done") {
                handlers.onDone?.();
            }
        });
    }
}

function parseSseEvent(raw) {
    const lines = raw.split("\n");
    const eventLine = lines.find((line) => line.startsWith("event:"));
    const dataLine = lines.find((line) => line.startsWith("data:"));

    if (!eventLine || !dataLine) {
        return null;
    }

    try {
        return {
            type: eventLine.replace("event:", "").trim(),
            data: JSON.parse(dataLine.replace("data:", "").trim())
        };
    } catch {
        return null;
    }
}
