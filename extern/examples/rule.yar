rule Veloc : Tag1 {
    strings:
        $s1 = {4c d3 bf 40 24}
        $s2 = {80 48 40 02}
    condition:
        $s1 and $s2
}

