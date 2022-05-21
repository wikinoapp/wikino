# frozen_string_literal: true

class SetupInitialSchema < ActiveRecord::Migration[7.0]
  def change
    create_table :users, id: :uuid do |t|
      t.string :auth0_user_id, null: false
      t.datetime :deleted_at
      t.timestamps
    end
    add_index :users, :auth0_user_id, unique: true

    create_table :notes, id: :uuid do |t|
      t.references :user, null: false, foreign_key: true, type: :uuid
      t.citext :title, null: false, default: ""
      t.string :content_type, null: false
      t.uuid :content_id, null: false
      t.datetime :modified_at, null: false
      t.timestamps
    end
    add_index :notes, %i[user_id title], unique: true
    add_index :notes, :created_at
    add_index :notes, :modified_at

    create_table :note_contents, id: :uuid do |t|
      t.references :user, null: false, foreign_key: true, type: :uuid
      t.references :note, null: false, foreign_key: true, type: :uuid, index: {unique: true}
      t.citext :body, null: false, default: ""
      t.text :body_html, null: false, default: ""
      t.timestamps
    end
    add_index :note_contents, %i[user_id note_id]

    create_table :archived_note_contents, id: :uuid do |t|
      t.references :user, null: false, foreign_key: true, type: :uuid
      t.references :note, null: false, foreign_key: true, type: :uuid
      t.citext :body, null: false, default: ""
      t.text :body_html, null: false, default: ""
      t.timestamps
    end
    add_index :archived_note_contents, %i[user_id note_id]

    create_table :links, id: :uuid do |t|
      t.references :note, null: false, foreign_key: true, type: :uuid
      t.references :target_note, null: false, foreign_key: {to_table: :notes}, type: :uuid
      t.timestamps
    end
    add_index :links, %i[note_id target_note_id], unique: true
    add_index :links, :created_at
  end
end
