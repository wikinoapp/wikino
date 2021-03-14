# frozen_string_literal: true

class ReHelloWorld < ActiveRecord::Migration[6.0]
  def change
    add_column :notes, :modified_at, :datetime
    add_column :users, :signed_up_at, :datetime, null: false
    add_column :users, :deleted_at, :datetime

    create_table :email_confirmations, id: :uuid do |t|
      t.citext :email, null: false
      t.integer :event, null: false
      t.string :token, null: false
      t.datetime :expires_at, null: false
      t.timestamps null: false
    end
    add_index :email_confirmations, :token, unique: true

    create_table :access_tokens, id: :uuid do |t|
      t.references :user, null: false, foreign_key: true, type: :uuid, index: { unique: true }
      t.string :token, null: false
      t.timestamps null: false
    end
    add_index :access_tokens, :token, unique: true
  end
end
