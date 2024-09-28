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
  has_many :draft_notes, dependent: :restrict_with_exception, foreign_key: :editor_id, inverse_of: :editor
  has_many :notes, dependent: :restrict_with_exception, foreign_key: :author_id, inverse_of: :author
  has_many :note_editorships, dependent: :restrict_with_exception, foreign_key: :editor_id, inverse_of: :editor
  has_many :notebook_memberships, dependent: :restrict_with_exception, foreign_key: :member_id, inverse_of: :member
  has_many :notebooks, through: :notebook_memberships
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

  sig { params(note: Note).returns(T.any(Note::PrivateCollectionProxy, Note::PrivateAssociationRelation)) }
  def notes_except(note)
    note.new_record? ? notes : notes.where.not(id: note.id)
  end

  sig { returns(Notebook::PrivateRelation) }
  def viewable_notebooks
    Notebook
      .left_joins(:memberships)
      .merge(
        Notebook.visibility_public.or(
          NotebookMembership.where(member: self)
        )
      )
      .distinct
  end

  sig { returns(Notebook::PrivateAssociationRelation) }
  def last_note_modified_notebooks
    notebooks.merge(
      notebook_memberships
        .order(NotebookMembership.arel_table[:last_note_modified_at].desc.nulls_last)
        .order(NotebookMembership.arel_table[:joined_at].desc)
    )
  end

  sig { returns(Note::PrivateAssociationRelation) }
  def last_modified_notes
    space.not_nil!.notes.joins(:editorships).merge(
      note_editorships.order(NoteEditorship.arel_table[:last_note_modified_at].desc)
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
    notebook_memberships.find_by(notebook:)&.role_admin? == true
  end

  sig { params(email_confirmation: EmailConfirmation).void }
  def run_after_email_confirmation_success!(email_confirmation:)
    return unless email_confirmation.succeeded?

    if email_confirmation.deserialized_event == EmailConfirmationEvent::EmailUpdate
      update!(email: email_confirmation.email)
    end

    nil
  end

  sig { params(notebook: Notebook).returns(Note) }
  def create_initial_note!(notebook:)
    notes.initial.where(notebook:).first_or_create!(
      space:,
      title: nil,
      body: "",
      body_html: "",
      linked_note_ids: [],
      modified_at: Time.current
    )
  end

  sig { params(note: Note).returns(DraftNote) }
  def find_or_create_draft_note!(note:)
    draft_notes.create_with(
      space:,
      notebook: note.notebook,
      title: note.title,
      body: note.body,
      body_html: note.body_html,
      linked_note_ids: note.linked_note_ids,
      modified_at: Time.zone.now
    ).find_or_create_by!(note:)
  rescue ActiveRecord::RecordNotUnique
    retry
  end

  sig { params(note: Note).void }
  def destroy_draft_note!(note:)
    draft_notes.where(note:).destroy_all

    nil
  end

  sig { params(notebook: Notebook, title: String).returns(Note) }
  def create_linked_note!(notebook:, title:)
    notes.where(notebook:, title:).first_or_create!(
      space:,
      body: "",
      body_html: "",
      linked_note_ids: [],
      modified_at: Time.zone.now
    )
  end
end
