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
  has_many :note_editors, dependent: :restrict_with_exception
  has_many :notebook_members, dependent: :restrict_with_exception
  has_many :notebooks, through: :notebook_members
  has_many :sessions, dependent: :restrict_with_exception
  has_one :user_password, dependent: :restrict_with_exception

  delegate :identifier, to: :space, prefix: :space

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

  T::Sig::WithoutRuntime.sig { returns(Notebook::PrivateRelation) }
  def viewable_notebooks
    Notebook
      .left_joins(:notebook_members)
      .merge(
        Notebook.visibility_public.or(
          NotebookMember.where(user_id: self.id)
        )
      )
      .distinct
  end

  T::Sig::WithoutRuntime.sig { returns(Notebook::PrivateRelation) }
  def last_note_modified_notebooks
    notebooks.merge(
      notebook_members
        .order(NotebookMember.arel_table[:last_note_modified_at].desc.nulls_last)
        .order(NotebookMember.arel_table[:joined_at].desc)
    )
  end

  T::Sig::WithoutRuntime.sig { returns(Note::PrivateRelation) }
  def last_modified_notes
    space.notes.joins(:note_editors).merge(
      note_editors.order(NoteEditor.arel_table[:last_note_modified_at].desc)
    )
  end

  sig { params(notebook: Notebook).returns(T::Boolean) }
  def can_view_notebook?(notebook:)
    viewable_notebooks.where(id: notebook.id).exists?
  end

  sig { params(notebook: Notebook).returns(T::Boolean) }
  def can_update_notebook?(notebook:)
    notebooks.where(id: notebook.id).exists?
  end

  sig { params(notebook: Notebook).returns(T::Boolean) }
  def can_destroy_notebook?(notebook:)
    notebook_members.find_by(notebook:)&.role&.admin? == true
  end

  sig { params(email_confirmation: EmailConfirmation).void }
  def run_after_email_confirmation_success!(email_confirmation:)
    return unless email_confirmation.succeeded?

    if email_confirmation.deserialized_event == EmailConfirmationEvent::EmailUpdate
      update!(email: email_confirmation.email)
    end

    nil
  end
end
