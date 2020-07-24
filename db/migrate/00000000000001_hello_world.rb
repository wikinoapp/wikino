# frozen_string_literal: true

class HelloWorld < ActiveRecord::Migration[6.0]
  def change
    enable_extension(:pgcrypto) unless extension_enabled?("pgcrypto")
    enable_extension(:citext) unless extension_enabled?("citext")

    create_table :users, id: :uuid do |t|
      t.citext :email, null: false
      t.string :encrypted_password, null: false
      t.string :reset_password_token
      t.datetime :reset_password_sent_at
      t.datetime :remember_created_at
      t.string :confirmation_token
      t.datetime :confirmed_at
      t.datetime :confirmation_sent_at
      t.string :unconfirmed_email # Only if using reconfirmable
      t.timestamps null: false
    end
    add_index :users, :email, unique: true
    add_index :users, :reset_password_token, unique: true
    add_index :users, :confirmation_token, unique: true

    create_table :notes, id: :uuid do |t|
      t.references :user, null: false, foreign_key: true, type: :uuid
      t.citext :title, null: false, default: ""
      t.text :body, null: false, default: ""
      t.timestamps null: false
    end
    add_index :notes, %i(user_id title), unique: true
    add_index :notes, :created_at
    add_index :notes, :updated_at

    create_table :references, id: :uuid do |t|
      t.references :note, null: false, foreign_key: true, type: :uuid
      t.references :referencing_note, null: false, foreign_key: { to_table: :notes }, type: :uuid
      t.timestamps null: false
    end
    add_index :references, %i(note_id referencing_note_id), unique: true
    add_index :references, :created_at
  end
end
