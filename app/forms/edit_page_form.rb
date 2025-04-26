# typed: strict
# frozen_string_literal: true

class EditPageForm < ApplicationForm
  include ActiveModel::Validations::Callbacks

  include FormConcerns::PageTitleValidatable

  sig { returns(T.nilable(SpaceMemberRecord)) }
  attr_accessor :space_member_record

  sig { returns(T.nilable(PageRecord)) }
  attr_accessor :page_record

  attribute :topic_number, :integer
  attribute :title, :string
  attribute :body, :string, default: ""

  before_validation :convert_nil_to_empty_string

  validates :space_member_record, presence: true
  validates :page_record, presence: true
  validates :topic, presence: true
  validate :title_uniqueness

  sig { returns(T.nilable(TopicRecord)) }
  def topic
    selectable_topics.find_by(number: topic_number)
  end

  sig { returns(T.any(TopicRecord::PrivateRelation, TopicRecord::PrivateCollectionProxy)) }
  def selectable_topics
    return TopicRecord.none if space_member_record.nil?

    space_member_record.not_nil!.topic_records
  end

  sig { returns(T::Boolean) }
  def autofocus_title?
    title.blank?
  end

  sig { returns(T::Boolean) }
  def autofocus_body?
    !autofocus_title?
  end

  sig { void }
  private def convert_nil_to_empty_string
    self.body = "" if body.nil?
  end

  sig { void }
  private def title_uniqueness
    return if topic.nil?

    other_page = topic.not_nil!.page_records.where.not(id: page_record.not_nil!.id).find_by(title:)

    if other_page
      edit_page_path = "/s/#{topic.not_nil!.space_record.not_nil!.identifier}/pages/#{other_page.number}/edit"
      errors.add(:title, I18n.t("forms.errors.models.edit_page_form.uniqueness_html", edit_page_path:))
    end
  end
end
