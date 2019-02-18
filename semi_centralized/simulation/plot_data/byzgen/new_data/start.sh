#!/bin/bash

python parser.py deterlab.csv 1 | sort -t',' -k1 -n > deterlab_parsed.csv
#python parser.py byzgen_11.csv 1 | sort -t',' -k1 -n > parsed/parsed_11.csv
#python parser.py byzgen_12.csv 1 | sort -t',' -k1 -n > parsed/parsed_12.csv
#python parser.py byzgen_21.csv 1 | sort -t',' -k1 -n > parsed/parsed_21.csv
#python parser.py byzgen_22.csv 1 | sort -t',' -k1 -n > parsed/parsed_22.csv
#python parser.py byzgen_31.csv 1 | sort -t',' -k1 -n > parsed/parsed_31.csv
#python parser.py byzgen_32.csv 1 | sort -t',' -k1 -n > parsed/parsed_32.csv
#python parser.py byzgen_41.csv 1 | sort -t',' -k1 -n > parsed/parsed_41.csv
#python parser.py byzgen_42.csv 1 | sort -t',' -k1 -n > parsed/parsed_42.csv
#python parser.py byzgen_51.csv 1 | sort -t',' -k1 -n > parsed/parsed_51.csv
#python parser.py byzgen_52.csv 1 | sort -t',' -k1 -n > parsed/parsed_52.csv
#python parser.py byzgen_61.csv 1 | sort -t',' -k1 -n > parsed/parsed_61.csv
#python parser.py byzgen_62.csv 1 | sort -t',' -k1 -n > parsed/parsed_62.csv
#python parser.py byzgen_71.csv 1 | sort -t',' -k1 -n > parsed/parsed_71.csv
#python parser.py byzgen_72.csv 1 | sort -t',' -k1 -n > parsed/parsed_72.csv
#python parser.py byzgen_81.csv 1 | sort -t',' -k1 -n > parsed/parsed_81.csv
#python parser.py byzgen_82.csv 1 | sort -t',' -k1 -n > parsed/parsed_82.csv

#python parser.py byzgen_11.csv 2 | tail -r > parsed/p_nonblock_11.csv
#python parser.py byzgen_12.csv 2 | tail -r > parsed/p_nonblock_12.csv
#python parser.py byzgen_21.csv 2 | tail -r > parsed/p_nonblock_21.csv
#python parser.py byzgen_22.csv 2 | tail -r > parsed/p_nonblock_22.csv
#python parser.py byzgen_31.csv 2 | tail -r > parsed/p_nonblock_31.csv
#python parser.py byzgen_32.csv 2 | tail -r > parsed/p_nonblock_32.csv
#python parser.py byzgen_41.csv 2 | tail -r > parsed/p_nonblock_41.csv
#python parser.py byzgen_42.csv 2 | tail -r > parsed/p_nonblock_42.csv
#python parser.py byzgen_51.csv 2 | tail -r > parsed/p_nonblock_51.csv
#python parser.py byzgen_52.csv 2 | tail -r > parsed/p_nonblock_52.csv
#python parser.py byzgen_61.csv 2 | tail -r > parsed/p_nonblock_61.csv
#python parser.py byzgen_62.csv 2 | tail -r > parsed/p_nonblock_62.csv
#python parser.py byzgen_71.csv 2 | tail -r > parsed/p_nonblock_71.csv
#python parser.py byzgen_72.csv 2 | tail -r > parsed/p_nonblock_72.csv
#python parser.py byzgen_81.csv 2 | tail -r > parsed/p_nonblock_81.csv
#python parser.py byzgen_82.csv 2 | tail -r > parsed/p_nonblock_82.csv

#python combiner.py parsed/parsed_11.csv counts/count_11.csv > parsed/combined_11.csv
#python combiner.py parsed/parsed_12.csv counts/count_12.csv > parsed/combined_12.csv
#python combiner.py parsed/parsed_21.csv counts/count_21.csv > parsed/combined_21.csv
#python combiner.py parsed/parsed_22.csv counts/count_22.csv > parsed/combined_22.csv
#python combiner.py parsed/parsed_31.csv counts/count_31.csv > parsed/combined_31.csv
#python combiner.py parsed/parsed_32.csv counts/count_32.csv > parsed/combined_32.csv
#python combiner.py parsed/parsed_41.csv counts/count_41.csv > parsed/combined_41.csv
#python combiner.py parsed/parsed_42.csv counts/count_42.csv > parsed/combined_42.csv
#python combiner.py parsed/parsed_51.csv counts/count_51.csv > parsed/combined_51.csv
#python combiner.py parsed/parsed_52.csv counts/count_52.csv > parsed/combined_52.csv
#python combiner.py parsed/parsed_61.csv counts/count_61.csv > parsed/combined_61.csv
#python combiner.py parsed/parsed_62.csv counts/count_62.csv > parsed/combined_62.csv
#python combiner.py parsed/parsed_71.csv counts/count_71.csv > parsed/combined_71.csv
#python combiner.py parsed/parsed_72.csv counts/count_72.csv > parsed/combined_72.csv
#python combiner.py parsed/parsed_81.csv counts/count_81.csv > parsed/combined_81.csv
#python combiner.py parsed/parsed_82.csv counts/count_82.csv > parsed/combined_82.csv
