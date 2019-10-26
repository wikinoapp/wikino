# frozen_string_literal: true

class SetupInitialSchema < ActiveRecord::Migration[6.0]
  def change
    enable_extension(:pgcrypto) unless extension_enabled?("pgcrypto")
    enable_extension(:citext) unless extension_enabled?("citext")

    create_table :users, id: :uuid do |t|
      t.timestamps null: false
      t.datetime :deleted_at
      t.citext :email, null: false
    end
    add_index :users, :deleted_at
    add_index :users, :email, unique: true

    create_table :teams, id: :uuid do |t|
      t.timestamps null: false
      t.datetime :deleted_at
      t.citext :subdomain, null: false
      t.string :name, null: false
    end
    add_index :teams, :deleted_at
    add_index :teams, :subdomain, unique: true

    create_table :projects, id: :uuid do |t|
      t.timestamps null: false
      t.datetime :deleted_at
      t.references :team, null: false, foreign_key: true, type: :uuid
      t.string :urlname, null: false
      t.string :name, null: false
    end
    add_index :projects, :deleted_at
    add_index :projects, :urlname, unique: true

    create_table :notes, id: :uuid do |t|
      t.timestamps null: false
      t.references :team, null: false, foreign_key: true, type: :uuid
      t.references :project, null: false, foreign_key: true, type: :uuid
      t.references :creator, null: false, foreign_key: { to_table: :users }, type: :uuid
      t.integer :number, null: false
      t.text :body
    end
    add_index :notes, %i(team_id number), unique: true

    create_table :team_members, id: :uuid do |t|
      t.timestamps null: false
      t.datetime :deleted_at
      t.references :team, null: false, foreign_key: true, type: :uuid
      t.references :user, null: false, foreign_key: true, type: :uuid
      t.citext :username, null: false
      t.string :name
    end
    add_index :team_members, :deleted_at
    add_index :team_members, %i(team_id user_id), unique: true
    add_index :team_members, %i(team_id username), unique: true

    create_table :project_members, id: :uuid do |t|
      t.timestamps null: false
      t.references :team, null: false, foreign_key: true, type: :uuid
      t.references :user, null: false, foreign_key: true, type: :uuid
      t.references :project, null: false, foreign_key: true, type: :uuid
      t.references :team_member, null: false, foreign_key: true, type: :uuid
    end
    add_index :project_members, %i(project_id team_member_id), unique: true

    create_table :oauth_providers, id: :uuid do |t|
      t.timestamps null: false
      t.references :user, null: false, foreign_key: true, type: :uuid
      t.integer :name, null: false
      t.string :uid, null: false
      t.string :token, null: false
      t.integer :token_expires_at
    end
    add_index :oauth_providers, %i(name uid), unique: true

    create_table :edges, id: :uuid do |t|
      t.timestamps null: false
      t.references :note, null: false, foreign_key: true, type: :uuid
      t.references :target_note, null: false, foreign_key: { to_table: :notes }, type: :uuid
    end
    add_index :edges, %i(note_id target_note_id), unique: true

    create_table :tags, id: :uuid do |t|
      t.timestamps null: false
      t.references :team, null: false, foreign_key: true, type: :uuid
      t.references :project, null: false, foreign_key: true, type: :uuid
      t.citext :name, null: false
    end
    add_index :tags, %i(project_id name), unique: true

    create_table :taggings, id: :uuid do |t|
      t.timestamps null: false
      t.references :team, null: false, foreign_key: true, type: :uuid
      t.references :project, null: false, foreign_key: true, type: :uuid
      t.references :note, null: false, foreign_key: true, type: :uuid
      t.references :tag, null: false, foreign_key: true, type: :uuid
    end
    add_index :taggings, %i(note_id tag_id), unique: true

    create_table :participants, id: :uuid do |t|
      t.timestamps null: false
      t.references :note, null: false, foreign_key: true, type: :uuid
      t.references :user, null: false, foreign_key: true, type: :uuid
    end
    add_index :participants, %i(note_id user_id), unique: true
  end
end
