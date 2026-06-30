# Accelerating Go: How we beat GNU Grep and reached 7.3 GB/s without a single line of Assembly

Many people think Go is a typical language for enterprise, form building, and boring microservices. And they are somewhat right. But unlike their perspective, we understand what hides under the hood if you dig a little deeper.

A very common task: exact search of multiple strings in a huge file. Who hasn't done this? We constantly use it for logging and analysis. For example, GNU `grep` is a great program. It has regex, word variations, and speed. So, what am I talking about? Ah yes... You haven't seen real speed yet.

Today, we are going to search for a dictionary of **50,000 unique keys** in a log file of exactly **1 Gigabyte**. And we will do it in Go.

---

### 1. Naive approach: Writing the file loop

First, let's sketch out the core. We will read the file not line-by-line (which kills the garbage collector), but in fat 10 Megabyte chunks using `bufio.Scanner`.

```go
func (m *Matcher) MatchReader(reader io.Reader) {
    scanner := bufio.NewScanner(reader)
    buf := make([]byte, 10*1024*1024)
    scanner.Buffer(buf, 10*1024*1024)

    for scanner.Scan() {
        lineBytes := scanner.Bytes()
        // ... substring search
    }
}
```

### 2. Setting up output and callbacks
To make our library universal, we won't write `fmt.Println` directly in the core. We will make an elegant Callback:

```go
type MatchCallback func(lineNum int, bytePos int, pattern []byte, lineContent []byte)
```
Now, upon every match, we will pass the line number and the found word to the outside.

### 3. Setting up parameters (Two-byte hash)
How do we search for 50,000 words simultaneously? Checking each word in a loop is death (we'd get 50 Terabytes of scanning). 
Write an Aho-Corasick tree? Long, complicated, and, running ahead—unnecessary.

We take the **first 2 bytes** of each search word, convert them into a `uint16`, and create an array of 65,536 buckets.
```go
type Matcher struct {
    buckets [65536][][]byte
}
```
Upon initialization (which takes a laughable 2 ms), we distribute the 50,000 words into these buckets. On average, there is **less than one word** per bucket!

### 4. Measuring the first version (Single thread, no unsafe)

In the loop, we simply take a 2-byte window and look into the `buckets[idx]` bucket. If it's not empty, we compare the remaining word via `bytes.Equal`.

Running the benchmark:
```text
BenchmarkMatcher-32      100      10927238 ns/op       959.60 MB/s
```
**~1 GB/s**. Good, but not enough. We want to squeeze all the juice out of the hardware!

---

### 5. The dark side of Go: Adding `unsafe` and multithreading

Now buckle up. We remove the safety checks of the Go compiler.

Instead of `idx := uint16(line[i]) | uint16(line[i+1])<<8`, which causes an array bounds check on every byte, we write:
```go
ptr := unsafe.Pointer(&data[0])
idx := *(*uint16)(unsafe.Pointer(uintptr(ptr) + i))
```
**What happens from the Assembly (ASM) perspective?** 
The Go compiler takes the hint and collapses this piece into *one single* assembly memory read instruction (like `MOVZX`), executed in one clock cycle!

Next—**CPU Cache**. We add `hasPattern [65536]bool` to the structure. This array weighs exactly **64 Kilobytes**—it perfectly, byte-for-byte, fits into the ultra-fast L1 cache of modern processors. We no longer touch RAM at all!

And finally—**Multithreading with smart queues**. 
We stream the file, slice it by the last newline `\n` (so as not to cut keys in half at chunk boundaries), and throw the pieces into `taskChan`. Workers process them in parallel, incrementing the result via atomic counters: `atomic.AddInt64(&matchesCount, 1)`.

### 6. Measuring and staying in shock

Running our new pure in-memory benchmark (`go test -bench . -benchmem`):
```text
goos: windows
goarch: amd64
pkg: grep/deanon
cpu: AMD Ryzen 9 7950X3D 16-Core Processor          
BenchmarkMatcher-32              100      10927238 ns/op       959.60 MB/s      10547080 B/op        126 allocs/op
BenchmarkMatcherParallel-32       97      14240700 ns/op      7363.23 MB/s      62197060 B/op       1169 allocs/op
```
**7.36 Gigabytes per second!** At the same time, the algorithm practically doesn't touch the garbage collector.

Out of curiosity, we take the native C GNU `grep` (v3.0 with Aho-Corasick support) and sic it on our 1GB log file (NVMe SSD).

### 7. Final comparison

Below are the results of a real scan of the `target.log` file of **1 Gigabyte** (search with `-F -f` flag for grep and `MatchReaderParallel` for our Go code).

#### Results Table (Dictionary of 10,000 keys)
| Tool | Threads | Scan Time (1 GB) | Throughput |
| :--- | :---: | :---: | :---: |
| **GNU Grep 3.0** | 1 (Single) | ~0.62 sec | ~1.6 GB/s |
| **Our Go (Base)** | 1 (Single) | ~0.95 sec | ~1.0 GB/s |
| **Our Go (Unsafe)** | 32 (Multi) | **0.29 sec** | **~3.4 GB/s** 🏆 |

#### Results Table (Dictionary of 50,000 keys)
The larger the dictionary, the more classic search suffers due to the growth of data structures. But our hash array has a fixed size (64 KB), so the speed hardly drops!

| Tool | Threads | Scan Time (1 GB) | Throughput |
| :--- | :---: | :---: | :---: |
| **GNU Grep 3.0** | 1 (Single) | 1.39 sec | ~0.7 GB/s |
| **Our Go (Base)** | 1 (Single) | 1.10 sec | ~0.9 GB/s |
| **Our Go (Unsafe)** | 32 (Multi) | **0.32 sec** | **~3.1 GB/s** 🏆 |

*(Note: On a real file, we hit a bottleneck at 3.1 GB/s, as this is the physical read speed limit of our SSD drive. In RAM, as seen from the benchmarks, the algorithm is capable of 7.3 GB/s).*

### Reflections: Why do we need "Aho-Corasick"?
The standard Aho-Corasick algorithm for multi-search builds a huge transition graph in RAM. For 50k words, that's hundreds of thousands of pointers. Running through this graph, you constantly catch Cache Misses. Our "stupid" approach with a 2-byte index array and direct access via `unsafe` puts the filter directly into the CPU L1 cache and destroys any trees.

### Conclusions
In our core, **there is not a single line written manually in Assembly** (we also didn't use full-fledged SIMD via C-bindings). We stayed within the standard Go tooling.

But if you study the tool you work with, you can work miracles. In C/C++ it would have been much harder and not necessarily faster: managing buffer pools, cross-platform work with channels and goroutines—in Go, this is done out of the box. We operate directly on the hardware (via `unsafe`), but at the same time completely and safely manage high-level queues and multithreading.

Don't be afraid to look under the hood!
