# typed: false
# frozen_string_literal: true

class InitialTables < ActiveRecord::Migration[7.1]
  def change
    create_table :email_confirmations, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.string :email, null: false
      t.integer :event, null: false
      t.string :code, null: false
      t.timestamp :started_at, null: false
      t.timestamp :succeeded_at
      t.timestamps

      t.index :started_at
      t.index %i[email code], unique: true
    end

    create_table :spaces, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.citext :identifier, index: {unique: true}, null: false
      t.string :name, null: false
      t.integer :plan, null: false
      t.timestamp :joined_at, null: false
      t.timestamp :discarded_at, index: true
      t.timestamps
    end

    create_table :users, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.references :space, foreign_key: true, null: false, type: :uuid
      t.string :email, null: false
      t.citext :atname, null: false
      t.integer :role, null: false
      t.string :name, default: "", null: false
      t.string :description, default: "", null: false
      t.integer :locale, null: false
      t.string :time_zone, null: false
      t.integer :sign_in_count, default: 0, null: false
      t.timestamp :current_signed_in_at
      t.timestamp :last_signed_in_at
      t.timestamp :joined_at, null: false
      t.timestamp :discarded_at, index: true
      t.timestamps

      t.index %i[space_id email], unique: true
      t.index %i[space_id atname], unique: true
    end

    create_table :user_passwords, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.references :space, foreign_key: true, null: false, type: :uuid
      t.references :user, foreign_key: true, index: {unique: true}, null: false, type: :uuid
      t.string :password_digest, null: false
      t.timestamps
    end

    create_table :sessions, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.references :space, foreign_key: true, null: false, type: :uuid
      t.references :user, foreign_key: true, null: false, type: :uuid
      t.string :token, index: {unique: true}, null: false
      t.string :ip_address, default: "", null: false
      t.string :user_agent, default: "", null: false
      t.datetime :signed_in_at, null: false
      t.timestamps
    end

    create_table :notebooks, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.references :space, foreign_key: true, null: false, type: :uuid
      t.citext :identifier, null: false
      t.string :name, default: "", null: false
      t.string :description, default: "", null: false
      t.integer :visibility, null: false
      t.timestamp :discarded_at, index: true

      t.index %i[space_id identifier], unique: true
    end

    create_table :notebook_members, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.references :space, foreign_key: true, null: false, type: :uuid
      t.references :notebook, foreign_key: true, null: false, type: :uuid
      t.references :user, foreign_key: true, null: false, type: :uuid
      t.string :role, null: false

      t.index %i[notebook_id user_id], unique: true
    end

    create_table :notes, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.references :space, foreign_key: true, null: false, type: :uuid
      t.references :notebook, foreign_key: true, null: false, type: :uuid
      t.references :author, foreign_key: {to_table: :users}, null: false, type: :uuid
      t.integer :number, null: false
      t.citext :title, default: "", null: false
      t.citext :body, default: "", null: false
      t.text :body_html, default: "", null: false
      t.datetime :modified_at, null: false
      t.timestamps

      t.index %i[space_id number], unique: true
      t.index %i[notebook_id title], unique: true
      t.index %i[notebook_id created_at]
      t.index %i[notebook_id modified_at]
    end

    create_table :note_revisions, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.references :space, foreign_key: true, null: false, type: :uuid
      t.references :note, foreign_key: true, null: false, type: :uuid
      t.references :editor, foreign_key: {to_table: :users}, null: false, type: :uuid
      t.citext :title, default: "", null: false
      t.citext :body, null: false, default: ""
      t.text :body_html, null: false, default: ""
      t.timestamps
    end

    create_table :note_links, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.references :source_note, foreign_key: {to_table: :notes}, null: false, type: :uuid
      t.references :target_note, foreign_key: {to_table: :notes}, null: false, type: :uuid
      t.timestamps

      t.index %i[source_note_id target_note_id], unique: true
    end
  end
end
