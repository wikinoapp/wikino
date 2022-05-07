# frozen_string_literal: true

class CreateDoorkeeperTables < ActiveRecord::Migration[7.0]
  def change
    create_table :oauth_applications, id: :uuid do |t|
      t.string  :name,               null: false
      t.string  :uid,                null: false
      t.string  :secret,             null: false

      # Remove `null: false` if you are planning to use grant flows
      # that doesn't require redirect URI to be used during authorization
      # like Client Credentials flow or Resource Owner Password.
      t.text    :redirect_uri,       null: false
      t.string  :scopes,             null: false, default: ''
      t.boolean :confidential,       null: false, default: true
      t.boolean :skip_authorization, null: false, default: false
      t.timestamps                   null: false
    end
    add_index :oauth_applications, :uid, unique: true

    create_table :oauth_application_members, id: :uuid do |t|
      t.references :application, null: false, foreign_key: { to_table: :oauth_applications }, type: :uuid
      t.references :user,        null: false, foreign_key: true, type: :uuid
      t.timestamps
    end
    add_index :oauth_application_members, %i(application_id user_id), unique: true

    create_table :oauth_access_grants, id: :uuid do |t|
      t.references :resource_owner,  null: false, foreign_key: { to_table: :users }, type: :uuid
      t.references :application,     null: false, foreign_key: { to_table: :oauth_applications }, type: :uuid
      t.string   :token,             null: false
      t.integer  :expires_in,        null: false
      t.text     :redirect_uri,      null: false
      t.string   :scopes,            null: false, default: ''
      t.datetime :revoked_at
      t.datetime :created_at,        null: false
    end
    add_index :oauth_access_grants, :token, unique: true

    create_table :oauth_access_tokens, id: :uuid do |t|
      t.references :resource_owner, foreign_key: { to_table: :users }, type: :uuid

      # Remove `null: false` if you are planning to use Password
      # Credentials Grant flow that doesn't require an application.
      t.references :application,    null: false, foreign_key: { to_table: :oauth_applications }, type: :uuid

      # If you use a custom token generator you may need to change this column
      # from string to text, so that it accepts tokens larger than 255
      # characters. More info on custom token generators in:
      # https://github.com/doorkeeper-gem/doorkeeper/tree/v3.0.0.rc1#custom-access-token-generator
      #
      # t.text :token, null: false
      t.string :token, null: false

      t.string   :refresh_token
      t.integer  :expires_in
      t.string   :scopes

      # The authorization server MAY issue a new refresh token, in which case
      # *the client MUST discard the old refresh token* and replace it with the
      # new refresh token. The authorization server MAY revoke the old
      # refresh token after issuing a new refresh token to the client.
      # @see https://datatracker.ietf.org/doc/html/rfc6749#section-6
      #
      # Doorkeeper implementation: if there is a `previous_refresh_token` column,
      # refresh tokens will be revoked after a related access token is used.
      # If there is no `previous_refresh_token` column, previous tokens are
      # revoked as soon as a new access token is created.
      #
      # Comment out this line if you want refresh tokens to be instantly
      # revoked after use.
      t.string   :previous_refresh_token, null: false, default: ""

      t.datetime :revoked_at
      t.datetime :created_at, null: false
    end
    add_index :oauth_access_tokens, :token, unique: true
    add_index :oauth_access_tokens, :refresh_token, unique: true

    # Uncomment below to ensure a valid reference to the resource owner's table
    # add_foreign_key :oauth_access_grants, <model>, column: :resource_owner_id
    # add_foreign_key :oauth_access_tokens, <model>, column: :resource_owner_id
  end
end
