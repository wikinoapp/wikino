# frozen_string_literal: true

class HelloWorld < ActiveRecord::Migration[6.0]
  def change
    enable_extension(:pgcrypto) unless extension_enabled?("pgcrypto")
    enable_extension(:citext) unless extension_enabled?("citext")

    create_table :users, id: :uuid do |t|
      t.citext :email, null: false
      t.string :access_token, null: false
      t.datetime :signed_up_at, null: false
      t.datetime :deleted_at
      t.timestamps null: false
    end
    add_index :users, :email, unique: true
    add_index :users, :access_token, unique: true

    create_table :email_confirmations, id: :uuid do |t|
      t.citext :email, null: false
      t.integer :event, null: false
      t.string :token, null: false
      t.datetime :expires_at, null: false
      t.timestamps null: false
    end
    add_index :email_confirmations, :token, unique: true

    create_table :notes, id: :uuid do |t|
      t.references :user, null: false, foreign_key: true, type: :uuid
      t.citext :title, null: false, default: ""
      t.text :body, null: false, default: ""
      t.text :body_html, null: false, default: ""
      t.string :cover_image_url
      t.datetime :modified_at
      t.timestamps null: false
    end
    add_index :notes, %i(user_id title), unique: true
    add_index :notes, :created_at
    add_index :notes, :updated_at

    create_table :links, id: :uuid do |t|
      t.references :note, null: false, foreign_key: true, type: :uuid
      t.references :target_note, null: false, foreign_key: { to_table: :notes }, type: :uuid
      t.timestamps null: false
    end
    add_index :links, %i(note_id target_note_id), unique: true
    add_index :links, :created_at
  end
end
