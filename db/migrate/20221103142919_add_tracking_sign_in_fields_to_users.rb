# typed: true
# frozen_string_literal: true

class AddTrackingSignInFieldsToUsers < ActiveRecord::Migration[7.0]
  def change
    add_column :users, :sign_in_count, :integer, default: 0, null: false
    add_column :users, :current_signed_in_at, :datetime
    add_column :users, :last_signed_in_at, :datetime
  end
end
