# typed: false
# frozen_string_literal: true

class AddDiscardedAtToPages < ActiveRecord::Migration[7.1]
  disable_ddl_transaction!

  def change
    add_column :pages, :discarded_at, :datetime
    add_index :pages, :discarded_at, algorithm: :concurrently
  end
end
