# Multi-precision Arithmetic

This package implements a form of fixed multi-precision integer arithmetic that
focuses on only a few operations. The payback for the fixed nature and limited
scope is no dynamic allocation and copy semantics.

The operations of interest include:

* Multiplication by a uint16
* Division and remainder of a multi-precision number and a uint16
* Modulus of two multi-precision numbers
* Computation of $a^n \mod m$ where $a$ and $m$ are multi-precision and $n$ is
  `uint64`

Each kind of integer implemented has a fixed number of 64-bit components that
are each used to hold 48 bits of the value of interest. This gives us 16 bits of
headroom for the multiplication operation and allows the clean implementation
of, for example, 288 bit integers (288 = 6 * 48).