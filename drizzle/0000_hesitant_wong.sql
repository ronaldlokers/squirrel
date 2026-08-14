CREATE TABLE "identities" (
	"id" bigserial PRIMARY KEY NOT NULL,
	"person_id" bigint NOT NULL,
	"transport" text NOT NULL,
	"external_id" text NOT NULL,
	"created_at" timestamp with time zone DEFAULT now() NOT NULL
);
--> statement-breakpoint
CREATE TABLE "items" (
	"id" bigserial PRIMARY KEY NOT NULL,
	"transport" text NOT NULL,
	"external_id" text,
	"conversation_id" text,
	"sender_id" text,
	"person_id" bigint,
	"raw_text" text NOT NULL,
	"payload" jsonb NOT NULL,
	"received_at" timestamp with time zone NOT NULL,
	"inserted_at" timestamp with time zone DEFAULT now() NOT NULL
);
--> statement-breakpoint
CREATE TABLE "people" (
	"id" bigserial PRIMARY KEY NOT NULL,
	"handle" text NOT NULL,
	"created_at" timestamp with time zone DEFAULT now() NOT NULL,
	CONSTRAINT "people_handle_unique" UNIQUE("handle")
);
--> statement-breakpoint
ALTER TABLE "identities" ADD CONSTRAINT "identities_person_id_people_id_fk" FOREIGN KEY ("person_id") REFERENCES "public"."people"("id") ON DELETE no action ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "items" ADD CONSTRAINT "items_person_id_people_id_fk" FOREIGN KEY ("person_id") REFERENCES "public"."people"("id") ON DELETE no action ON UPDATE no action;--> statement-breakpoint
CREATE UNIQUE INDEX "identities_transport_external_id_key" ON "identities" USING btree ("transport","external_id");--> statement-breakpoint
CREATE UNIQUE INDEX "items_transport_external_id_key" ON "items" USING btree ("transport","external_id") WHERE "items"."external_id" is not null;--> statement-breakpoint
CREATE INDEX "items_received_at_idx" ON "items" USING btree ("received_at");