# typed: false
# frozen_string_literal: true

class AddPinnedAtToPages < ActiveRecord::Migration[7.1]
  disable_ddl_transaction!

  def change
    add_column :pages, :pinned_at, :timestamp
    add_index :pages, %i[space_id pinned_at], algorithm: :concurrently
  end
end
