# typed: false
# frozen_string_literal: true

class UpdateSchema202402032 < ActiveRecord::Migration[7.1]
  def change
    StrongMigrations.disable_check(:change_column_null_postgresql)
    change_column_null :topic_memberships, :created_at, false
    change_column_null :topic_memberships, :updated_at, false
    change_column_null :topics, :created_at, false
    change_column_null :topics, :updated_at, false
  end
end
