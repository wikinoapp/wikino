# typed: strict
# frozen_string_literal: true

class EditSuggestionPageRecord < ApplicationRecord
  self.table_name = "edit_suggestion_pages"

  belongs_to :space_record, foreign_key: :space_id
  belongs_to :edit_suggestion_record, foreign_key: :edit_suggestion_id
  belongs_to :page_record, foreign_key: :page_id, optional: true
  belongs_to :page_revision_record, foreign_key: :page_revision_id, optional: true
  belongs_to :latest_revision_record, foreign_key: :latest_revision_id, class_name: "EditSuggestionPageRevisionRecord"

  has_many :revision_records, foreign_key: :edit_suggestion_page_id, class_name: "EditSuggestionPageRevisionRecord", dependent: :restrict_with_exception

  sig { returns(T::Boolean) }
  def new_page?
    page_id.nil? && page_revision_id.nil?
  end

  sig { returns(T::Boolean) }
  def existing_page?
    !new_page?
  end

  sig { returns(T::Boolean) }
  def title_changed?
    return true if new_page?

    page_revision_record.not_nil!.title != latest_revision_record.not_nil!.title
  end

  sig { returns(T::Boolean) }
  def body_changed?
    return true if new_page?

    page_revision_record.not_nil!.body != latest_revision_record.not_nil!.body
  end

  sig { returns(T::Boolean) }
  def has_changes?
    title_changed? || body_changed?
  end

  # 最新リビジョンの情報をプロキシするメソッド
  sig { returns(String) }
  def title
    latest_revision_record.not_nil!.title
  end

  sig { returns(String) }
  def body
    latest_revision_record.not_nil!.body
  end

  sig { returns(String) }
  def body_html
    latest_revision_record.not_nil!.body_html
  end
end
