#!/usr/bin/env perl

use v5.38;

my $M_PI  = 3.14159265358979323846;

my $ary = [ [1,2],[3,4] ];

my $r = naive_perl_dct2d($ary);

1; 

sub naive_perl_dct2d {
    my $vector = shift;
    my $N      = scalar(@$vector);
    my $fact   = $M_PI/$N;
    my ($temp, $result);

    for (my $x = 0; $x < $N; $x++) {
        for (my $i = 0; $i < $N; $i++) {
            my $sum = 0;
            for (my $j = 0; $j < $N; $j++) {
                $sum += $vector->[$x]->[$j] * cos(($j+0.5)*$i*$fact);
            }
            $temp->[$x]->[$i] = $sum;
        }
    }

    for (my $y = 0; $y < $N; $y++) {
        for (my $i = 0; $i < $N; $i++) {
            my $sum = 0;
            for (my $j = 0; $j < $N; $j++) {
                $sum += $temp->[$j]->[$y] * cos(($j+0.5)*$i*$fact);
            }
            $result->[$i]->[$y] = $sum;
        }
    }
    return $result;
}
