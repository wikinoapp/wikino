# typed: strict
# frozen_string_literal: true

class User < ApplicationRecord
  include Discard::Model

  include ModelConcerns::Viewable

  ATNAME_FORMAT = /\A[A-Za-z0-9_]+\z/
  ATNAME_MIN_LENGTH = 1
  ATNAME_MAX_LENGTH = 20

  enum :locale, {
    ViewerLocale::En.serialize => 0,
    ViewerLocale::Ja.serialize => 1
  }, prefix: true

  belongs_to :space, optional: true
  has_many :draft_pages, dependent: :restrict_with_exception, foreign_key: :editor_id, inverse_of: :editor
  has_many :page_editorships, dependent: :restrict_with_exception, foreign_key: :editor_id, inverse_of: :editor
  has_many :pages, through: :page_editorships
  has_many :topic_memberships, dependent: :restrict_with_exception, foreign_key: :member_id, inverse_of: :member
  has_many :topics, through: :topic_memberships
  has_many :space_members, dependent: :restrict_with_exception
  has_many :active_space_members, -> { active }, class_name: "SpaceMember", dependent: :restrict_with_exception, inverse_of: :user
  has_many :spaces, through: :space_members
  has_many :active_spaces, class_name: "Space", through: :active_space_members, source: :space
  has_many :user_sessions, dependent: :restrict_with_exception
  has_one :user_password, dependent: :restrict_with_exception

  delegate :identifier, :name, to: :space, prefix: :space

  sig do
    params(
      email: String,
      atname: String,
      password: String,
      locale: ViewerLocale,
      time_zone: String,
      current_time: ActiveSupport::TimeWithZone
    ).returns(User)
  end
  def self.create_initial_user!(email:, atname:, password:, locale:, time_zone:, current_time:)
    user = create!(
      email:,
      atname:,
      name: "",
      description: "",
      locale: locale.serialize,
      time_zone:,
      joined_at: current_time
    )
    user.create_user_password!(password:)

    user
  end

  sig { override.returns(T::Boolean) }
  def signed_in?
    true
  end

  sig { override.returns(ViewerLocale) }
  def locale
    ViewerLocale.deserialize(read_attribute(:locale))
  end

  sig { params(page: Page).returns(T.any(Page::PrivateCollectionProxy, Page::PrivateAssociationRelation)) }
  def pages_except(page)
    page.new_record? ? pages : pages.where.not(id: page.id)
  end

  sig { override.returns(Topic::PrivateRelation) }
  def viewable_topics
    Topic
      .left_joins(:memberships)
      .merge(
        Topic.visibility_public.or(
          TopicMembership.where(member: self)
        )
      )
      .distinct
  end

  sig { returns(Topic::PrivateAssociationRelation) }
  def last_page_modified_topics
    topics.merge(
      topic_memberships
        .order(TopicMembership.arel_table[:last_page_modified_at].desc.nulls_last)
        .order(TopicMembership.arel_table[:joined_at].desc)
    )
  end

  sig { returns(Page::PrivateAssociationRelation) }
  def last_modified_pages
    space.not_nil!.pages.joins(:editorships).merge(
      page_editorships.order(PageEditorship.arel_table[:last_page_modified_at].desc)
    )
  end

  sig { override.params(page: Page).returns(T::Boolean) }
  def can_view_page?(page:)
    page.topic.not_nil!.visibility_public? ||
      space_members.where(space: page.space).active.exists?
  end

  sig { params(topic: Topic).returns(T::Boolean) }
  def can_view_topic?(topic:)
    viewable_topics.where(id: topic.id).exists?
  end

  sig { params(topic: Topic).returns(T::Boolean) }
  def can_update_topic?(topic:)
    topics.where(id: topic.id).exists?
  end

  sig { params(topic: Topic).returns(T::Boolean) }
  def can_destroy_topic?(topic:)
    topic_memberships.find_by(topic:)&.role_admin? == true
  end

  sig { override.params(page: Page).returns(T::Boolean) }
  def can_trash_page?(page:)
    active_spaces.where(id: page.space_id).exists?
  end

  sig { params(topic: Topic).returns(T::Boolean) }
  def joined_topic?(topic:)
    topics.include?(topic)
  end

  sig { params(topic_ids: T::Array[T::Wikino::DatabaseId]).returns(T::Boolean) }
  def joined_all_topics?(topic_ids:)
    joined_topic_ids = topics.pluck(:id)
    topic_ids - joined_topic_ids == []
  end

  sig { params(email_confirmation: EmailConfirmation).void }
  def run_after_email_confirmation_success!(email_confirmation:)
    return unless email_confirmation.succeeded?

    if email_confirmation.deserialized_event == EmailConfirmationEvent::EmailUpdate
      update!(email: email_confirmation.email)
    end

    nil
  end

  sig { params(page: Page).returns(DraftPage) }
  def find_or_create_draft_page!(page:)
    draft_pages.create_with(
      space:,
      topic: page.topic,
      title: page.title,
      body: page.body,
      body_html: page.body_html,
      linked_page_ids: page.linked_page_ids,
      modified_at: Time.zone.now
    ).find_or_create_by!(page:)
  rescue ActiveRecord::RecordNotUnique
    retry
  end

  sig { params(page: Page).void }
  def destroy_draft_page!(page:)
    draft_pages.where(page:).destroy_all

    nil
  end

  sig { params(topic: Topic, title: String).returns(Page) }
  def create_linked_page!(topic:, title:)
    page = space.not_nil!.pages.where(topic:, title:).first_or_create!(
      space:,
      body: "",
      body_html: "",
      linked_page_ids: [],
      modified_at: Time.zone.now
    )
    page_editorships.where(page:).first_or_create!(space:, last_page_modified_at: page.modified_at)

    page
  end
end
