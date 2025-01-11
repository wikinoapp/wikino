# typed: false
# frozen_string_literal: true

class RenameArchivedAtToTrashedAtOnPages < ActiveRecord::Migration[7.1]
  def change
    StrongMigrations.disable_check(:rename_column)

    rename_column :pages, :archived_at, :trashed_at
  end
end
