CREATE TABLE IF NOT EXISTS pfc
(
    data date NOT NULL,
    proteins numeric NOT NULL,
    fats numeric NOT NULL,
    carbs numeric NOT NULL,
    CONSTRAINT pfc_pkey PRIMARY KEY (data)
);