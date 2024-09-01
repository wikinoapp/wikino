# typed: strict
# frozen_string_literal: true

class User < ApplicationRecord
  extend T::Sig

  include Discard::Model

  ATNAME_FORMAT = /\A[A-Za-z0-9_]+\z/
  ATNAME_MIN_LENGTH = 2
  ATNAME_MAX_LENGTH = 20

  enum :role, {
    UserRole::Owner.serialize => 0
  }, prefix: true

  enum :locale, {
    UserLocale::En.serialize => 0,
    UserLocale::Ja.serialize => 1
  }, prefix: true

  belongs_to :space
  has_many :notes, dependent: :restrict_with_exception, foreign_key: :author_id
  has_many :note_editorships, dependent: :restrict_with_exception, foreign_key: :editor_id
  has_many :list_memberships, dependent: :restrict_with_exception, foreign_key: :member_id
  has_many :lists, through: :list_memberships
  has_many :sessions, dependent: :restrict_with_exception
  has_one :user_password, dependent: :restrict_with_exception

  delegate :identifier, :name, to: :space, prefix: :space

  sig do
    params(
      email: String,
      atname: String,
      password: String,
      locale: UserLocale,
      time_zone: String,
      current_time: ActiveSupport::TimeWithZone
    ).returns(User)
  end
  def self.create_initial_user!(email:, atname:, password:, locale:, time_zone:, current_time:)
    user = create!(
      email:,
      atname:,
      role: UserRole::Owner.serialize,
      name: "",
      description: "",
      locale: locale.serialize,
      time_zone:,
      joined_at: current_time
    )
    user.create_user_password!(space: user.space, password:)

    user
  end

  sig { returns(UserLocale) }
  def deserialized_locale
    UserLocale.deserialize(locale)
  end

  T::Sig::WithoutRuntime.sig { params(note: Note).returns(Note::PrivateRelation) }
  def notes_except(note)
    note.new_record? ? notes : notes.where.not(id: note.id)
  end

  T::Sig::WithoutRuntime.sig { returns(List::PrivateRelation) }
  def viewable_lists
    List
      .left_joins(:memberships)
      .merge(
        List.visibility_public.or(
          ListMembership.where(member: self)
        )
      )
      .distinct
  end

  T::Sig::WithoutRuntime.sig { returns(List::PrivateRelation) }
  def last_note_modified_lists
    lists.merge(
      list_memberships
        .order(ListMembership.arel_table[:last_note_modified_at].desc.nulls_last)
        .order(ListMembership.arel_table[:joined_at].desc)
    )
  end

  T::Sig::WithoutRuntime.sig { returns(Note::PrivateRelation) }
  def last_modified_notes
    space.notes.joins(:note_editorships).merge(
      note_editorships.order(NoteEditorship.arel_table[:last_note_modified_at].desc)
    )
  end

  sig { params(list: List).returns(T::Boolean) }
  def can_view_list?(list:)
    viewable_lists.where(id: list.id).exists?
  end

  sig { params(list: List).returns(T::Boolean) }
  def can_update_list?(list:)
    lists.where(id: list.id).exists?
  end

  sig { params(list: List).returns(T::Boolean) }
  def can_destroy_list?(list:)
    list_memberships.find_by(list:)&.role&.admin? == true
  end

  sig { params(email_confirmation: EmailConfirmation).void }
  def run_after_email_confirmation_success!(email_confirmation:)
    return unless email_confirmation.succeeded?

    if email_confirmation.deserialized_event == EmailConfirmationEvent::EmailUpdate
      update!(email: email_confirmation.email)
    end

    nil
  end

  sig { params(list: List).returns(Note) }
  def create_initial_note!(list:)
    notes.initial.where(list:).first_or_create!(
      space:,
      title: nil,
      body: "",
      body_html: "",
      modified_at: Time.current
    )
  end

  sig { params(list: List, title: String).returns(Note) }
  def create_linked_note!(list:, title:)
    notes.where(list:, title:).first_or_create!(
      space:,
      body: "",
      body_html: "",
      modified_at: Time.current
    )
  end
end
