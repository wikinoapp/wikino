# typed: false
# frozen_string_literal: true

class ValidateEditSuggestionsForeignKeys < ActiveRecord::Migration[8.0]
  def change
    validate_foreign_key :edit_suggestions, :spaces
    validate_foreign_key :edit_suggestions, :topics
    validate_foreign_key :edit_suggestions, :users

    validate_foreign_key :edit_suggestion_pages, :spaces
    validate_foreign_key :edit_suggestion_pages, :edit_suggestions
    validate_foreign_key :edit_suggestion_pages, :pages

    validate_foreign_key :edit_suggestion_comments, :spaces
    validate_foreign_key :edit_suggestion_comments, :edit_suggestions
    validate_foreign_key :edit_suggestion_comments, :users
  end
end
