# typed: false
# frozen_string_literal: true

class AddUniqueIndexToAtnameAndEmailOnUsers < ActiveRecord::Migration[7.1]
  disable_ddl_transaction!

  def change
    add_index :users, :email, unique: true, algorithm: :concurrently
    add_index :users, :atname, unique: true, algorithm: :concurrently
  end
end
