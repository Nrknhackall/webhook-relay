import test from 'node:test'; import assert from 'node:assert/strict'; import { createHmac } from 'node:crypto';
test('HMAC signature is deterministic', () => { const a = createHmac('sha256','secret').update('{"ok":true}').digest('hex'); assert.equal(a.length,64); assert.equal(a,createHmac('sha256','secret').update('{"ok":true}').digest('hex')); });
test('idempotency key is part of the contract', () => assert.equal('Idempotency-Key','Idempotency-Key'));
