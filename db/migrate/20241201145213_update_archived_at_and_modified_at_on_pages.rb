# typed: false
# frozen_string_literal: true

class UpdateArchivedAtAndModifiedAtOnPages < ActiveRecord::Migration[7.1]
  def change
    StrongMigrations.disable_check(:rename_column)

    rename_column :pages, :archived_at, :trashed_at
    change_column_null :pages, :modified_at, true
  end
end
