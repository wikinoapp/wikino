# typed: strict
# frozen_string_literal: true

class SpaceMemberRecord < ApplicationRecord
  self.table_name = "space_members"

  belongs_to :space_record, foreign_key: :space_id
  belongs_to :user_record, foreign_key: :user_id
  has_many :topic_member_records,
    dependent: :restrict_with_exception,
    foreign_key: :space_member_id,
    inverse_of: :space_member_record
  has_many :topic_records,
    foreign_key: :topic_id,
    through: :topic_member_records
  has_many :draft_page_records,
    dependent: :restrict_with_exception,
    foreign_key: :space_member_id,
    inverse_of: :space_member_record
  has_many :page_editor_records,
    dependent: :restrict_with_exception,
    foreign_key: :space_member_id,
    inverse_of: :space_member_record

  delegate :locale, :time_zone, to: :user_record, prefix: true

  scope :active, -> { where(active: true) }
  scope :inactive, -> { where(active: false) }

  sig { params(page_record: PageRecord).void }
  def destroy_draft_page!(page_record:)
    draft_page_records.where(page_record:).destroy_all

    nil
  end

  sig { returns(PageRecord::PrivateAssociationRelation) }
  def last_modified_pages
    space_record.not_nil!.page_records.joins(:page_editor_records).merge(
      page_editor_records.order(PageEditorRecord.arel_table[:last_page_modified_at].desc)
    )
  end

  sig { returns(T.any(TopicRecord::PrivateAssociationRelation, TopicRecord::PrivateRelation)) }
  def joined_topic_records
    topic_records.kept
  end
end
