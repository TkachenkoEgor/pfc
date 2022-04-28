CREATE TABLE IF NOT EXISTS pfc (
	date date NOT NULL,
	proteins numeric NOT NULL,
	fats numeric NOT NULL,
	carbs numeric NOT NULL,
	CONSTRAINT pfc_pkey PRIMARY KEY (date)
);
