import Redis from "ioredis"

const client = new Redis("rediss://default:AVqcAAIncDI4ZTEzZTI3NGY2Yzk0MWNjYThjZDc5ZWM1MzRkYmIzZXAyMjMxOTY@national-mammal-23196.upstash.io:6379");
await client.set('foo', 'bar');

export default client;