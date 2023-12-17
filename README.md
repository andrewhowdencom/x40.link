# s3k.link

The [Skink](https://en.wikipedia.org/wiki/Skink) short link service.

[going away]: https://firebase.google.com/support/dynamic-links-faq

## Understanding this work

This project functions as a demonstration of work. In the future, I will likely cannibalize it for the
[Practical Introduction to Observability](h4n.link/pito). In it, you can see:

### 🧪 Test Driven Development

Take a look around at the files
[suffixed with _test](https://github.com/search?q=repo%3Aandrewhowdencom%2Fs3k.link+path%3A_test.go&type=code). You'll
see the popular "table-driven test" format, with many tests being invoked in parallel to ensure fast execution and concurrency safety.
You'll also see the occasional
[benchmark](https://github.com/search?q=repo%3Aandrewhowdencom%2Fs3k.link+path%3A_test.go+Benchmark&type=code), as well
as well as some
[testable examples](https://github.com/search?q=repo%3Aandrewhowdencom%2Fs3k.link+path%3A_test.go+Example&type=code).
You can learn more on [the go website](https://go.dev/blog/examples). Some tests even validate concurrency via
[the go race detector](https://go.dev/blog/race-detector) and `go test -race`!

### 📈 Algorithmic Complexity

You'll see [different implementations of the same problem] —
[finding a URL in a set](https://github.com/andrewhowdencom/s3k.link/tree/main/storage/memory). There is the
naive, [linear implementation](https://github.com/andrewhowdencom/s3k.link/blob/main/storage/memory/linear_search.go),
a [binary search implementation](https://github.com/andrewhowdencom/s3k.link/blob/main/storage/memory/binary_search.go),
and the one we'd
actually use — a very [simple hashmap](https://github.com/andrewhowdencom/s3k.link/blob/main/storage/memory/hash_table.go).
There are even benchmarks to
[validate their performance!](https://github.com/andrewhowdencom/s3k.link/blob/main/storage/storage_test.go#L87-L149)

### ✍️ Helpful commit messages

Read the [commit history to understand my thinking](https://github.com/andrewhowdencom/s3k.link/commits/main/) while
writing each unit of work. You can see how the thinking has changed over time! You can read more
[about why I think this is so important](https://medium.com/@andrewhowdencom/anatomy-of-a-good-commit-message-acd9c4490437)

### ❓ Remaining Work

Quite a bit of work remains in this project before it becomes "production-ready!" For example,

1. The definition of the project as a container
2. The deployment on a cloud infrastructure provider (e.g., Kubernetes)
3. The configuration of a public cloud (e.g., Terraform)
4. Observability instrumentation (e.g., logs, metrics, traces, profiling)
5. Data Storage (e.g., Postgres, BoltDB, or DynamoDB)
6. Service Level Management (e.g., SLOs, SLAs)

I have worked with all of these technologies before; however, I have only a limited number of daily hours! Perhaps
you can [email me](mailto:hello@andrewhowden.com), and I can help you find evidence of what you're looking for.


## Development & Deployment

See [the development documentation](DEVELOPMENT.md) or the [deployment documentation](DEPLOYMENT.md)