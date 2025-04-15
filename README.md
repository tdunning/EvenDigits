# Fast Scanner for OEIS Sequence A068994

The programs here implement a fast scanning technique designed to find powers of
two whose digits are all even. The technique is based on building a large sieve
that can eliminate most candidates without examination. Only the very few
elements that pass through the sieve need to be examined in depth.

These programs were able to extend the number of powers of two with no new
elements of A068994 to n < 10^14 running for about 10-12 hours on a single core
of a laptop computer.

## Construction of the Sieve

The sieve is constructed by noting that the low order digits of powers of two
must eventually enter a cycle since they are taken from a finite set. Many of
these lower order digits contain an odd digit. We can skip any power of two that
has these digits. About half of the remaining elements of the cycle follow an
element that would cause a carry when doubled which implies that the digit just
to the left of the digits we are examining would be odd. We can skip these
elements as well.

The following diagram illustrates the cycle for the least significant two digits
of powers of two. In this diagram, we start with $2^0$ which has 01 as the lower
two digits and then proceed to 02, 04, 08, 16 and so on. The 10 values with an
odd digit are shown in faded italic lettering. Those with all even digits which
are preceded by a value large enough to cause a carry are marked with a red
background and have no border. The values 01 and 02 are outside of the cycle and
are marked with an oval shape. The remaining 5 values (08, 24, 48, 64, and 88)
are the elements of the sieve; all powers to two not ending in these values can
be eliminated without examination.

![img.png](images/img.png)

The sieve for two digits is well known since at least 2002. But there is no need
to stop with two digits. The program `cycle/cycles.go` computes the content of
analogous sieves for any reasonable number of digits. The value in going to
longer cycles is that the fraction of values in the cycle that have to be
examined drops dramatically as more digits are used. For instance, at 13 digits,
the cycle of lower digits extends to 976,562,500 values, but only 112,846 need
to be evaluated compared to the 5/20 that need to be evaluated when considering
2 digits. This gives a speedup of roughly 2000x when we have $n > 30$.

## Scanning Candidate Values

Actually testing values of $2^n$ to see if all digits are even does not usually
require that the entire value be computed. This is good
because $2^{10^{14}} \approx 3 \times 10^{10^{13}}$ (that is, it has $10^{13}$
digits all together). Fortunately, for all values scanned so far, an odd digit
occurs in the least significant 46 digits. This means that computing just the 50
or 60 least significant digits will suffice to eliminate candidate values.

Since 60 digits of decimal precision requires only 200 bits, it will still take
an extended precision library, but it doesn't have to be very exotic.
Hand-rolling something using base-10 is possible, but very unlikely to be faster
than something that uses native arithmetic.

## Multi-threading The Search

This search process can be multi-threaded very easily since the search for each
repetition of the sieve is independent of every other. Further, once you have a
large cycle, examining the candidates takes a significant amount of time so the
mechanism for distributing work no longer matters to performance.

The code here distributes work to workers through a single channel. Assignments
are distributed in ascending order so that the work of computing the starting
point can be re-used by each worker.

Even without multi-threading, the system is very fast. Searching the
first $10^{14}$ values of $2^n$ took about 10 hours using a single core on my
laptop.

# Running the code

To run the search, you first need to find the cycle for some number of digits.
Once you have the sieve you want, you use that to run the search. There is a bit
of trade-off here. Using a larger cycle as the basis for the sieve makes the
search more efficient, but larger sieves use more memory and may not fit into
the cache. For a laptop, using 12 or 13 digits for the sieve seems about right.

## Generating a Sieve

To run this code, you start with the cycle generator.

```
% go run cycle/cycles.go
                                                                                   gain vs 
  digits  tail           cycle  exclude  maximal   last            gte5    even  brute force
       1     1               4     true     true      1               2       2       2.00
       2     2              20     true     true      2              15       5       4.00
       3     3             100     true     true      4              88      12       8.33
       4     4             500     true     true      8             470      30      16.67
       5     5           2,500     true     true     16           2,426      74      33.78
       6     6          12,500     true     true     32          12,315     185      67.57
       7     7          62,500     true     true     64          62,038     462     135.28
       8     8         312,500     true     true    128         311,344   1,156     270.33
       9     9       1,562,500     true     true    256       1,559,611   2,889     540.84
      10    10       7,812,500     true     true    512       7,805,279   7,221   1,081.91
      11    11      39,062,500     true     true  1,024      39,044,444  18,056   2,163.41
      12    12     195,312,500     true     true  2,048     195,267,361  45,139   4,326.91
      13    13     976,562,500     true     true  4,096     976,449,654 112,846   8,653.94
      14    14   4,882,812,500     true     true  8,192   4,882,530,389 282,111  17,308.13
      15    15  24,414,062,500     true     true 16,384  24,413,357,228 705,272  34,616.52
```

A side effect of running this program is the creation of a number of JSON files
that each encode a sieve. Here, for instance, is the file for the 2 digit sieve:

```
{
  "Mask": 100,
  "Order": 2,
  "Length": 20,
  "Leadin": 2,
  "EvenItems": 5,
  "Gain": 4,
  "Cycle": [8, 24, 48, 64, 88],
  "Index": [3, 6, 10, 11, 19]
}
```

The cycle generator uses Floyd's tortoise and hare algorithm to find the cycle
as well as the steps from $2^0$ to the first element of the cycle. All of the
values in the cycle are sorted. When the cycle is read into the search program,
the differences between successive indexes are used to step from one candidate
to the next. Note that the mask in the cycle file only defines the size of the
cycle and is different from the mask used in the search program where the mask
defines how many significant digits are used to disqualify candidate values.

## Commentary on Sieves

Elementary analysis of the product group formed by calculating $2^n \mod 10^k$
shows that the length of cycle in each sieve should be $4 \times 5^{k-1}$. If
the digits are randomly distributed, the fraction of values that pass the sieve
would be $2^{-k}$. Both of these are born out in the observed sieves. A
consequence of this is that this kind of sieve will never stop all candidates,
no matter how large $k$ gets.

## Running the Search

You can run a simplified search that only uses the 2-digit sieve using the
program `simple/scan-simple.go`. This allows the following options:

| Option    | Meaning                                                        |
|-----------|----------------------------------------------------------------|
| -verbose  | Provide progress information                                   |
| -limit n  | How many candidates to search. Use M, G, T, P, or E as desired |
| -digits d | How many digits to check for even digits                       |

This program is single-thread and can scan about 5M candidates per second.

The program `sieve/scan.go` is a more complex scanner. It allows a choice of how
many threads to use as well as selection of the sieve. By default, a 13-digit
sieve is used. The following options are allowed:

| Option     | Meaning                                                        |
|------------|----------------------------------------------------------------|
| -verbose   | Provide progress information                                   |
| -limit n   | How many candidates to search. Use M, G, T, P, or E as desired |
| -digits d  | How many digits to check for even digits                       |
| -threads t | How many threads to use to check candidates                    |
| -sieve s   | The name of a JSON file containing a sieve definition.         |

This scanner can scan about 10M candidates per second per thread with
`cycle-002.json` (the standard 2-digit sieve) but accelerates to 85M candidates
per second with `cycle-009.json` and to roughly 10G candidates per second per
thread with `cycle-013.json`.

# Results

Running many threads on an 18 core older server, this system was able to test
100P candidates using a 15 digit sieve (no additional solutions found). I expect
that using the sieve with $d$-digits to build the sieve with $d+1$-digits would
speed the cycle characterization up enough to build a 20 digit sieve which
should make the program run 32 times faster. That puts a 1E sample run into
view.

In earlier tests running on a single core of an older server, these were the
results using a 15 digit sieve. Per core, this machine is a bit slower than a
single ARM core on my laptop but the server has much more memory which becomes
important with the larger sieve basis. This is a run testing $10^15$ values. The
most recent versions also use the new mp package which puts pretty much all of
the extended precision numbers onto the stack instead of the heap. This results
in about half the memory usage.

```
dunning@host:~/EvenDigits$ go run sieve/scan.go -threads 1 -sieve cycle-015.json -limit 1P -verbose -digits 55
2025/03/23 00:39:14 Limit: 1000.0T
1 threads
2025/03/23 01:29:24 sender:   2048 (         5%, 1440.3 2950614.5) 56043.8 seconds remaining
2025/03/23 02:18:30 sender:   4096 (        10%, 1439.4 5896521.3) 53060.2 seconds remaining
2025/03/23 03:07:33 sender:   6144 (        15%, 1438.6 8839697.1) 50086.3 seconds remaining
2025/03/23 03:56:43 sender:   8192 (        20%, 1439.1 11789698.4) 47155.0 seconds remaining
2025/03/23 04:45:56 sender:  10240 (        25%, 1439.7 14743024.7) 44226.3 seconds remaining
2025/03/23 05:35:00 sender:  12288 (        30%, 1439.3 17686717.1) 41266.8 seconds remaining
2025/03/23 06:24:00 sender:  14336 (        35%, 1438.7 20626742.5) 38305.1 seconds remaining
2025/03/23 07:12:58 sender:  16384 (        40%, 1438.2 23564470.4) 35345.3 seconds remaining
2025/03/23 08:01:55 sender:  18432 (        45%, 1437.8 26501718.5) 32389.8 seconds remaining
2025/03/23 08:50:57 sender:  20480 (        50%, 1437.6 29443578.9) 29442.6 seconds remaining
2025/03/23 09:39:56 sender:  22528 (        55%, 1437.4 32382870.7) 26494.3 seconds remaining
2025/03/23 10:28:55 sender:  24576 (        60%, 1437.2 35321932.1) 23547.3 seconds remaining
2025/03/23 11:17:55 sender:  26624 (        65%, 1437.1 38262431.1) 20602.3 seconds remaining
2025/03/23 12:06:54 sender:  28672 (        70%, 1436.9 41201059.6) 17657.2 seconds remaining
2025/03/23 12:55:54 sender:  30720 (        75%, 1436.8 44140753.1) 14713.3 seconds remaining
2025/03/23 13:44:51 sender:  32768 (        80%, 1436.7 47078455.3) 11769.4 seconds remaining
2025/03/23 14:33:50 sender:  34816 (        85%, 1436.6 50017074.8) 8826.4 seconds remaining
2025/03/23 15:22:51 sender:  36864 (        90%, 1436.6 52958163.5) 5884.1 seconds remaining
2025/03/23 16:11:54 sender:  38912 (        95%, 1436.6 55901109.5) 2942.1 seconds remaining
2025/03/23 17:00:54 sender:  40960 (       100%, 1436.5 58840955.8) 0.0 seconds remaining
2025/03/23 17:00:54 sender: completed
2025/03/23 17:00:57 breaking 0
2025/03/23 17:00:57 exiting 0
2025/03/23 17:00:57 thread 0 result (max = 52)
16994150355.1 test/s, total time 58843.8 s
Limit: 1000000000000000
Tests: 28887941120
Gain over brute: 34616.520293.1
solutions = [1 2 3 6 11]
dunning@host:~/EvenDigits$ 
```

# Known Defects

1) The code as it stands has a problem running in multi-core mode that seems to
   be related to accidental sharing of non-thread-safe memory structures,
   particular those from the `math/big` library.
2) The code does not yet use the new extended precision library. That should
   resolve the multi-threading issues and may be faster than the `math/big`
   library.
3) Currently, outside of the extended precision math library the code has no
   unit tests which expose a risk that there might be remaining code errors.
4) The cycle detector can be made much simpler and faster because we know how
   many steps it takes to get into the cycle and we know that the cycle
   with $n+1$ digits is composed of 5 cycles with $n$ digits. That means we only
   have to examine the candidates from the $n$-digit cycles to find the roughly
   50% that survive as $n+1$-digit sieve values. The $n$-digit cycles will, of
   course, can be found faster by using the $n-1$-digit cycles. This will make
   finding larger cycles thousands of times faster.
5) Finally, memory usage seems anomalously high. For the largest sieve, the
   running program consumes 5-6GB of main storage. This seems excessive, but no
   profiling has been done yet to understand the source.