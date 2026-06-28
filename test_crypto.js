const { webcrypto } = require('crypto');
const { performance } = require('perf_hooks');

async function runBenchmark() {
    const sizes = [1, 10, 50, 100, 500, 1000]; // KB
    const results = [];

    // Generate a key
    const key = await webcrypto.subtle.generateKey(
        { name: 'AES-GCM', length: 256 },
        true,
        ['encrypt', 'decrypt']
    );

    const iv = webcrypto.getRandomValues(new Uint8Array(12));

    const { randomFillSync } = require('crypto');
    console.log("Size(KB),Time(ms)");

    for (const size of sizes) {
        // Create payload of `size` KB
        const data = new Uint8Array(size * 1024);
        randomFillSync(data); // Fill with random data to simulate real text

        // Warmup
        for(let i=0; i<10; i++) {
            await webcrypto.subtle.encrypt({ name: 'AES-GCM', iv }, key, data);
        }

        // Benchmark
        const iterations = 1000;
        const start = performance.now();
        for(let i=0; i<iterations; i++) {
            await webcrypto.subtle.encrypt({ name: 'AES-GCM', iv }, key, data);
        }
        const end = performance.now();
        
        const avgTime = (end - start) / iterations;
        console.log(`${size},${avgTime.toFixed(4)}`);
    }
}

runBenchmark().catch(console.error);
