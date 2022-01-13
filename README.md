# sequential-counter

Serverless sequential counter using gcloud storage and ifGenerationMatch attribute.

Probably will not be the perfect solution for high traffic scenario.

Work fine for several hundreds os invoices a day for example.

Initialization:

Create a bucket named counter

In the above bucker cretae a text file per counter with the initial number.
