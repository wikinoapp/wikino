# typed: false
# frozen_string_literal: true

class UpdateSchema202402031 < ActiveRecord::Migration[7.1]
  def change
    add_column :topic_memberships, :created_at, :datetime
    add_column :topic_memberships, :updated_at, :datetime
    add_column :topics, :created_at, :datetime
    add_column :topics, :updated_at, :datetime
  end
end
