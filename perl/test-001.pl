#!/usr/bin/env perl

use v5.38;

use Data::Printer;
use Math::DCT qw/dct dct1d dct2d idct1d idct2d/;

# DCT of 1D array
my $ary = [ [ 0.3181653197002592, 0.39066343796185155, 0.16102608753078032] ];
p($ary);
my $dct1d = dct($ary);
print "dct1d: \n";
p($dct1d);
print "\n";

# DCT of 2D array
$ary = [ [ 1, 2 ], [ 3, 4 ] ];
p($ary);
print "dct2d: \n";
my $dct2d = dct($ary);
p($dct2d);
print "\n";

# iDCT of 1D and 2D array
$ary = [ [ 1, 2, 3, 4, ] ];
p($ary);
print "idct1d: \n";
my $idct1d = idct1d($ary);
p($idct1d);
print "\n";
print "idct1d: \n";
my $idct2d = idct2d($ary);
p($idct2d);
print "\n";

$ary = [ [ 1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0, 11.0, 12.0, 13.0, 14.0, 15.0, 16.0, ] ];
p($ary);
print "dct2d: \n";
$dct2d = dct($ary);
p($dct2d);
print "\n";



