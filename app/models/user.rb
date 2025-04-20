# typed: true
# frozen_string_literal: true

class User < ApplicationModel
  include ModelConcerns::Viewable

  ATNAME_FORMAT = /\A[A-Za-z0-9_]+\z/
  # アットネームの最大文字数 (値に強い理由は無い)
  ATNAME_MAX_LENGTH = 20
  # 名前の最大文字数 (値に強い理由は無い)
  NAME_MAX_LENGTH = 30
  # 説明の最大文字数 (値に強い理由は無い)
  DESCRIPTION_MAX_LENGTH = 150

  sig { returns(T::Wikino::DatabaseId) }
  attr_accessor :database_id

  sig { returns(String) }
  attr_accessor :atname

  sig { returns(String) }
  attr_accessor :name

  sig { returns(String) }
  attr_accessor :description

  sig { override.returns(String) }
  attr_accessor :serialized_locale

  sig { override.returns(String) }
  attr_accessor :time_zone

  sig { override.returns(T::Boolean) }
  def signed_in?
    true
  end
end
