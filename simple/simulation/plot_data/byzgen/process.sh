#!/bin/bash

cat orig/byzgen_1.csv | awk -F',' '{print $41","$42","$116","$117","$26","$27","$86","$87","$56","$57}' >> byzgen_processed.csv
cat orig/byzgen_2.csv | awk -F',' '{print $41","$42","$116","$117","$26","$27","$86","$87","$56","$57}' >> byzgen_processed.csv
cat orig/byzgen_3.csv | awk -F',' '{print $41","$42","$116","$117","$26","$27","$86","$87","$56","$57}' >> byzgen_processed.csv
cat orig/byzgen_4.csv | awk -F',' '{print $41","$42","$116","$117","$26","$27","$86","$87","$56","$57}' >> byzgen_processed.csv
cat orig/byzgen_5.csv | awk -F',' '{print $41","$42","$116","$117","$26","$27","$86","$87","$56","$57}' >> byzgen_processed.csv
cat orig/byzgen_6.csv | awk -F',' '{print $41","$42","$116","$117","$26","$27","$86","$87","$56","$57}' >> byzgen_processed.csv
cat orig/byzgen_7.csv | awk -F',' '{print $41","$42","$116","$117","$26","$27","$86","$87","$56","$57}' >> byzgen_processed.csv
cat orig/byzgen_8.csv | awk -F',' '{print $41","$42","$116","$117","$26","$27","$86","$87","$56","$57}' >> byzgen_processed.csv
