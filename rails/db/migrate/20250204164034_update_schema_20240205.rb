# typed: false
# frozen_string_literal: true

class UpdateSchema20240205 < ActiveRecord::Migration[7.1]
  def change
    StrongMigrations.disable_check(:rename_table)
    rename_table :page_editorships, :page_editors
    rename_table :topic_memberships, :topic_members

    StrongMigrations.disable_check(:rename_column)
    rename_column :draft_pages, :editor_id, :space_member_id
    rename_column :page_editors, :editor_id, :space_member_id
    rename_column :page_revisions, :editor_id, :space_member_id
    rename_column :topic_members, :member_id, :space_member_id
  end
end
