# typed: false
# frozen_string_literal: true

class UpdateUserAndSpaceRelationship < ActiveRecord::Migration[7.1]
  disable_ddl_transaction!

  def change
    StrongMigrations.disable_check(:remove_column)
    remove_column :users, :role, :integer

    create_table :space_members, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.references :space, foreign_key: true, null: false, type: :uuid
      t.references :user, foreign_key: true, null: false, type: :uuid
      t.integer :role, null: false
      t.timestamp :joined_at, null: false
      t.timestamps

      t.index %i[space_id user_id], unique: true
    end

    StrongMigrations.disable_check(:rename_table)
    rename_table :sessions, :user_sessions
    remove_column :user_sessions, :space_id, :uuid

    remove_column :user_passwords, :space_id, :uuid

    remove_foreign_key :draft_pages, :users
    remove_foreign_key :page_editorships, :users
    remove_foreign_key :page_revisions, :users
    remove_foreign_key :topic_memberships, :users
  end
end
