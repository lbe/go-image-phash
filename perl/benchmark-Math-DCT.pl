#!/usr/bin/env perl

use v5.38;

use Benchmark qw(timethis);
use Math::DCT ':all';

my $iter = 1;
my $sz   = 32;
my @arrays;
push @arrays, [ map { rand(256) } ( 1 .. $sz * $sz ) ] foreach 1 .. 10;

my $dct;
my $fn_dct = sub() {
    foreach ( 1 .. $iter ) {
        $dct = dct2d( $arrays[ $iter % 10 ], $sz );
        #$d->add( $dct->[0] ) if $_ % 10 == 1;
    }
};

for (0 .. 4) {
    timethis(200_000, $fn_dct)
}
#
#timethis 20000:  1 wallclock secs ( 1.21 usr +  0.00 sys =  1.21 CPU) @ 16528.93/s (n=20000)
#timethis 20000:  1 wallclock secs ( 1.04 usr +  0.00 sys =  1.04 CPU) @ 19230.77/s (n=20000)
#timethis 20000:  1 wallclock secs ( 1.03 usr +  0.00 sys =  1.03 CPU) @ 19417.48/s (n=20000)
#timethis 20000:  1 wallclock secs ( 1.04 usr +  0.01 sys =  1.05 CPU) @ 19047.62/s (n=20000)
#timethis 20000:  1 wallclock secs ( 1.04 usr +  0.00 sys =  1.04 CPU) @ 19230.77/s (n=20000)
#
