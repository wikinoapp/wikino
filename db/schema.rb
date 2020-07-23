# This file is auto-generated from the current state of the database. Instead
# of editing this file, please use the migrations feature of Active Record to
# incrementally modify your database, and then regenerate this schema definition.
#
# This file is the source Rails uses to define your schema when running `rails
# db:schema:load`. When creating a new database, `rails db:schema:load` tends to
# be faster and is potentially less error prone than running all of your
# migrations from scratch. Old migrations may fail to apply correctly if those
# migrations use external dependencies or application code.
#
# It's strongly recommended that you check this file into your version control system.

ActiveRecord::Schema.define(version: 1) do

  # These are extensions that must be enabled in order to support this database
  enable_extension "citext"
  enable_extension "pgcrypto"
  enable_extension "plpgsql"

  create_table "notes", force: :cascade do |t|
    t.bigint "team_id", null: false
    t.bigint "project_id", null: false
    t.bigint "team_member_id", null: false
    t.bigint "number", null: false
    t.string "title", default: "", null: false
    t.text "body", default: "", null: false
    t.datetime "created_at", precision: 6, null: false
    t.datetime "updated_at", precision: 6, null: false
    t.index ["created_at"], name: "index_notes_on_created_at"
    t.index ["project_id", "title"], name: "index_notes_on_project_id_and_title", unique: true
    t.index ["project_id"], name: "index_notes_on_project_id"
    t.index ["team_id", "number"], name: "index_notes_on_team_id_and_number", unique: true
    t.index ["team_id"], name: "index_notes_on_team_id"
    t.index ["team_member_id"], name: "index_notes_on_team_member_id"
    t.index ["updated_at"], name: "index_notes_on_updated_at"
  end

  create_table "projects", force: :cascade do |t|
    t.bigint "team_id", null: false
    t.string "name", default: "", null: false
    t.datetime "created_at", precision: 6, null: false
    t.datetime "updated_at", precision: 6, null: false
    t.index ["team_id"], name: "index_projects_on_team_id"
  end

  create_table "references", force: :cascade do |t|
    t.bigint "note_id", null: false
    t.bigint "referencing_note_id", null: false
    t.datetime "created_at", precision: 6, null: false
    t.datetime "updated_at", precision: 6, null: false
    t.index ["created_at"], name: "index_references_on_created_at"
    t.index ["note_id", "referencing_note_id"], name: "index_references_on_note_id_and_referencing_note_id", unique: true
    t.index ["note_id"], name: "index_references_on_note_id"
    t.index ["referencing_note_id"], name: "index_references_on_referencing_note_id"
  end

  create_table "taggings", force: :cascade do |t|
    t.bigint "note_id", null: false
    t.bigint "tag_id", null: false
    t.datetime "created_at", precision: 6, null: false
    t.datetime "updated_at", precision: 6, null: false
    t.index ["created_at"], name: "index_taggings_on_created_at"
    t.index ["note_id", "tag_id"], name: "index_taggings_on_note_id_and_tag_id", unique: true
    t.index ["note_id"], name: "index_taggings_on_note_id"
    t.index ["tag_id"], name: "index_taggings_on_tag_id"
  end

  create_table "tags", force: :cascade do |t|
    t.bigint "project_id", null: false
    t.citext "name", null: false
    t.datetime "created_at", precision: 6, null: false
    t.datetime "updated_at", precision: 6, null: false
    t.index ["project_id", "name"], name: "index_tags_on_project_id_and_name", unique: true
    t.index ["project_id"], name: "index_tags_on_project_id"
    t.index ["updated_at"], name: "index_tags_on_updated_at"
  end

  create_table "team_members", force: :cascade do |t|
    t.bigint "user_id", null: false
    t.bigint "team_id", null: false
    t.string "name", default: "", null: false
    t.datetime "created_at", precision: 6, null: false
    t.datetime "updated_at", precision: 6, null: false
    t.index ["team_id"], name: "index_team_members_on_team_id"
    t.index ["user_id", "team_id"], name: "index_team_members_on_user_id_and_team_id", unique: true
    t.index ["user_id"], name: "index_team_members_on_user_id"
  end

  create_table "teams", force: :cascade do |t|
    t.citext "teamname", null: false
    t.string "name", default: "", null: false
    t.datetime "created_at", precision: 6, null: false
    t.datetime "updated_at", precision: 6, null: false
    t.index ["teamname"], name: "index_teams_on_teamname", unique: true
  end

  create_table "users", force: :cascade do |t|
    t.citext "email", null: false
    t.string "encrypted_password", null: false
    t.string "reset_password_token"
    t.datetime "reset_password_sent_at"
    t.datetime "remember_created_at"
    t.string "confirmation_token"
    t.datetime "confirmed_at"
    t.datetime "confirmation_sent_at"
    t.string "unconfirmed_email"
    t.datetime "created_at", precision: 6, null: false
    t.datetime "updated_at", precision: 6, null: false
    t.index ["confirmation_token"], name: "index_users_on_confirmation_token", unique: true
    t.index ["email"], name: "index_users_on_email", unique: true
    t.index ["reset_password_token"], name: "index_users_on_reset_password_token", unique: true
  end

  add_foreign_key "notes", "projects"
  add_foreign_key "notes", "team_members"
  add_foreign_key "notes", "teams"
  add_foreign_key "projects", "teams"
  add_foreign_key "references", "notes"
  add_foreign_key "references", "notes", column: "referencing_note_id"
  add_foreign_key "taggings", "notes"
  add_foreign_key "taggings", "tags"
  add_foreign_key "tags", "projects"
  add_foreign_key "team_members", "teams"
  add_foreign_key "team_members", "users"
end
