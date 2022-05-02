# This file is auto-generated from the current state of the database. Instead
# of editing this file, please use the migrations feature of Active Record to
# incrementally modify your database, and then regenerate this schema definition.
#
# This file is the source Rails uses to define your schema when running `bin/rails
# db:schema:load`. When creating a new database, `bin/rails db:schema:load` tends to
# be faster and is potentially less error prone than running all of your
# migrations from scratch. Old migrations may fail to apply correctly if those
# migrations use external dependencies or application code.
#
# It's strongly recommended that you check this file into your version control system.

ActiveRecord::Schema[7.0].define(version: 2021_00_00_000001) do
  # These are extensions that must be enabled in order to support this database
  enable_extension "citext"
  enable_extension "pgcrypto"
  enable_extension "plpgsql"

  create_table "access_tokens", id: :uuid, default: -> { "gen_random_uuid()" }, force: :cascade do |t|
    t.uuid "user_id", null: false
    t.string "token", null: false
    t.datetime "created_at", null: false
    t.datetime "updated_at", null: false
    t.index ["token"], name: "index_access_tokens_on_token", unique: true
    t.index ["user_id"], name: "index_access_tokens_on_user_id", unique: true
  end

  create_table "email_confirmations", id: :uuid, default: -> { "gen_random_uuid()" }, force: :cascade do |t|
    t.citext "email", null: false
    t.integer "event", null: false
    t.string "token", null: false
    t.datetime "expires_at", precision: nil, null: false
    t.datetime "created_at", null: false
    t.datetime "updated_at", null: false
    t.index ["token"], name: "index_email_confirmations_on_token", unique: true
  end

  create_table "links", id: :uuid, default: -> { "gen_random_uuid()" }, force: :cascade do |t|
    t.uuid "note_id", null: false
    t.uuid "target_note_id", null: false
    t.datetime "created_at", null: false
    t.datetime "updated_at", null: false
    t.index ["created_at"], name: "index_links_on_created_at"
    t.index ["note_id", "target_note_id"], name: "index_links_on_note_id_and_target_note_id", unique: true
    t.index ["note_id"], name: "index_links_on_note_id"
    t.index ["target_note_id"], name: "index_links_on_target_note_id"
  end

  create_table "notes", id: :uuid, default: -> { "gen_random_uuid()" }, force: :cascade do |t|
    t.uuid "user_id", null: false
    t.citext "title", default: "", null: false
    t.text "body", default: "", null: false
    t.text "body_html", default: "", null: false
    t.string "cover_image_url"
    t.datetime "created_at", null: false
    t.datetime "updated_at", null: false
    t.datetime "modified_at", precision: nil
    t.index ["created_at"], name: "index_notes_on_created_at"
    t.index ["updated_at"], name: "index_notes_on_updated_at"
    t.index ["user_id", "title"], name: "index_notes_on_user_id_and_title", unique: true
    t.index ["user_id"], name: "index_notes_on_user_id"
  end

  create_table "users", id: :uuid, default: -> { "gen_random_uuid()" }, force: :cascade do |t|
    t.citext "email", null: false
    t.string "encrypted_password", null: false
    t.string "reset_password_token"
    t.datetime "reset_password_sent_at", precision: nil
    t.datetime "remember_created_at", precision: nil
    t.string "confirmation_token"
    t.datetime "confirmed_at", precision: nil
    t.datetime "confirmation_sent_at", precision: nil
    t.string "unconfirmed_email"
    t.datetime "created_at", null: false
    t.datetime "updated_at", null: false
    t.datetime "signed_up_at", precision: nil, null: false
    t.datetime "deleted_at", precision: nil
    t.index ["confirmation_token"], name: "index_users_on_confirmation_token", unique: true
    t.index ["email"], name: "index_users_on_email", unique: true
    t.index ["reset_password_token"], name: "index_users_on_reset_password_token", unique: true
  end

  add_foreign_key "access_tokens", "users"
  add_foreign_key "links", "notes"
  add_foreign_key "links", "notes", column: "target_note_id"
  add_foreign_key "notes", "users"
end
