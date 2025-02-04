# typed: false
# frozen_string_literal: true

class UpdateUserAndSpaceRelationship2 < ActiveRecord::Migration[7.1]
  def change
    StrongMigrations.disable_check(:remove_column)
    remove_column :users, :space_id, :uuid

    StrongMigrations.disable_check(:add_foreign_key)
    add_foreign_key :draft_pages, :space_members, column: :editor_id
    add_foreign_key :page_editorships, :space_members, column: :editor_id
    add_foreign_key :page_revisions, :space_members, column: :editor_id
    add_foreign_key :topic_memberships, :space_members, column: :member_id
  end
end
