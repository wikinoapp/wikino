# typed: false
# frozen_string_literal: true

class InitialTables < ActiveRecord::Migration[7.1]
  def change
    create_table :email_confirmations, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.string :email, null: false
      t.string :event, null: false
      t.string :code, null: false
      t.timestamp :started_at, null: false
      t.timestamp :succeeded_at
      t.timestamps

      t.index :started_at
      t.index %i[email code], unique: true
    end

    create_table :profiles, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.string :profileable_type, null: false
      t.citext :atname, index: {unique: true}, null: false
      t.string :name, default: "", null: false
      t.string :description, default: "", null: false
      t.timestamp :joined_at, null: false
      t.timestamp :deleted_at
      t.timestamps
    end

    create_table :users, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.references :profile, foreign_key: true, index: {unique: true}, null: false, type: :uuid
      t.string :email, index: {unique: true}, null: false
      t.string :password_digest, null: false
      t.string :locale, null: false
      t.string :time_zone, null: false
      t.integer :sign_in_count, default: 0, null: false
      t.timestamp :current_signed_in_at
      t.timestamp :last_signed_in_at
      t.timestamp :signed_up_at, null: false
      t.timestamps
    end

    create_table :actors, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.references :user, foreign_key: true, index: {unique: true}, null: false, type: :uuid
      t.references :profile, foreign_key: true, index: {unique: true}, null: false, type: :uuid
      t.timestamps

      t.index %i[user_id profile_id], unique: true
    end

    create_table :teams, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.references :profile, foreign_key: true, index: {unique: true}, null: false, type: :uuid
    end

    create_table :team_memberships, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.references :team, foreign_key: true, null: false, type: :uuid
      t.references :user, foreign_key: true, null: false, type: :uuid

      t.index %i[team_id user_id], unique: true
    end

    create_table :projects, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.citext :identifier, index: {unique: true}, null: false
      t.string :name, default: "", null: false
    end

    create_table :project_memberships, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.references :project, foreign_key: true, null: false, type: :uuid
      t.references :user, foreign_key: true, null: false, type: :uuid

      t.index %i[project_id user_id], unique: true
    end


    create_table :notes, id: :uuid do |t|
      t.references :user, null: false, foreign_key: true, type: :uuid
      t.citext :title, null: false, default: ""
      t.datetime :modified_at, null: false
      t.timestamps
    end
    add_index :notes, %i[user_id title], unique: true
    add_index :notes, %i[user_id created_at]
    add_index :notes, %i[user_id modified_at]

    create_table :note_contents, id: :uuid do |t|
      t.references :user, null: false, foreign_key: true, type: :uuid
      t.references :note, null: false, foreign_key: true, type: :uuid
      t.citext :body, null: false, default: ""
      t.text :body_html, null: false, default: ""
      t.timestamps
    end
    add_index :note_contents, %i[user_id note_id], unique: true

    create_table :archived_note_contents, id: :uuid do |t|
      t.references :user, null: false, foreign_key: true, type: :uuid
      t.references :note, null: false, foreign_key: true, type: :uuid
      t.citext :body, null: false, default: ""
      t.text :body_html, null: false, default: ""
      t.timestamps
    end
    add_index :archived_note_contents, %i[user_id note_id]

    create_table :links, id: :uuid do |t|
      t.references :note, null: false, foreign_key: true, type: :uuid
      t.references :target_note, null: false, foreign_key: {to_table: :notes}, type: :uuid
      t.timestamps
    end
    add_index :links, %i[note_id target_note_id], unique: true
    add_index :links, :created_at
  end
end
