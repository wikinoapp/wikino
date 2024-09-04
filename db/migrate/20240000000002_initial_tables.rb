# typed: false
# frozen_string_literal: true

class InitialTables < ActiveRecord::Migration[7.1]
  def change
    create_table :email_confirmations, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.string :email, null: false
      t.integer :event, null: false
      t.string :code, index: {unique: true}, null: false
      t.timestamp :started_at, index: true, null: false
      t.timestamp :succeeded_at
      t.timestamps
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
      t.string :name, null: false
      t.string :description, null: false
      t.integer :locale, null: false
      t.string :time_zone, null: false
      t.timestamp :joined_at, null: false
      t.timestamp :discarded_at, index: true
      t.timestamps

      t.index %i[space_id email], unique: true
      t.index %i[space_id atname], unique: true
      t.index %i[space_id discarded_at]
    end

    create_table :user_passwords, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.references :space, foreign_key: true, null: false, type: :uuid
      t.references :user, foreign_key: true, index: {unique: true}, null: false, type: :uuid
      t.string :password_digest, null: false
      t.timestamps

      t.index %i[space_id user_id]
    end

    create_table :sessions, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.references :space, foreign_key: true, null: false, type: :uuid
      t.references :user, foreign_key: true, null: false, type: :uuid
      t.string :token, index: {unique: true}, null: false
      t.string :ip_address, null: false
      t.string :user_agent, null: false
      t.datetime :signed_in_at, null: false
      t.timestamps
    end

    create_table :lists, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.references :space, foreign_key: true, null: false, type: :uuid
      t.integer :number, null: false
      t.string :name, null: false
      t.string :description, null: false
      t.integer :visibility, null: false
      t.timestamp :discarded_at, index: true

      t.index %i[space_id number], unique: true
      t.index %i[space_id name], unique: true
      t.index %i[space_id discarded_at]
    end

    create_table :list_memberships, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.references :space, foreign_key: true, null: false, type: :uuid
      t.references :list, foreign_key: true, null: false, type: :uuid
      t.references :member, foreign_key: {to_table: :users}, null: false, type: :uuid
      t.integer :role, null: false
      t.timestamp :joined_at, null: false
      t.datetime :last_note_modified_at

      t.index %i[list_id member_id], unique: true
    end

    create_table :notes, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.references :space, foreign_key: true, null: false, type: :uuid
      t.references :list, foreign_key: true, null: false, type: :uuid
      t.references :author, foreign_key: {to_table: :users}, null: false, type: :uuid
      t.integer :number, null: false
      t.citext :title
      t.citext :body, null: false
      t.text :body_html, null: false
      t.string :linked_note_ids, array: true, index: {using: "gin"}, null: false
      t.datetime :modified_at, null: false
      t.datetime :published_at
      t.datetime :archived_at
      t.timestamps

      t.index %i[space_id number], unique: true
      t.index %i[space_id created_at]
      t.index %i[space_id modified_at]
      t.index %i[space_id published_at]
      t.index %i[space_id archived_at]
      t.index %i[list_id title], unique: true
    end

    create_table :note_editorships, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.references :space, foreign_key: true, null: false, type: :uuid
      t.references :note, foreign_key: true, null: false, type: :uuid
      t.references :editor, foreign_key: {to_table: :users}, null: false, type: :uuid
      t.datetime :last_note_modified_at, null: false
      t.timestamps

      t.index %i[note_id editor_id], unique: true
    end

    create_table :draft_notes, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.references :space, foreign_key: true, null: false, type: :uuid
      t.references :note, foreign_key: true, null: false, type: :uuid
      t.references :editor, foreign_key: {to_table: :users}, null: false, type: :uuid
      t.references :list, foreign_key: true, null: false, type: :uuid
      t.citext :title
      t.citext :body, null: false
      t.text :body_html, null: false
      t.string :linked_note_ids, array: true, index: {using: "gin"}, null: false
      t.timestamps

      t.index %i[editor_id note_id], unique: true
    end

    create_table :note_revisions, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.references :space, foreign_key: true, null: false, type: :uuid
      t.references :editor, foreign_key: {to_table: :users}, null: false, type: :uuid
      t.references :note, foreign_key: true, null: false, type: :uuid
      t.citext :body, null: false
      t.text :body_html, null: false
      t.timestamps
    end
  end
end
