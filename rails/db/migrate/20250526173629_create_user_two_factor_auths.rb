# typed: false
# frozen_string_literal: true

class CreateUserTwoFactorAuths < ActiveRecord::Migration[8.0]
  def change
    create_table :user_two_factor_auths, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.references :user, null: false, foreign_key: true, index: {unique: true}, type: :uuid
      t.string :secret, null: false
      t.boolean :enabled, null: false, default: false
      t.datetime :enabled_at
      t.string :recovery_codes, array: true, null: false, default: []
      t.timestamps
    end
  end
end
