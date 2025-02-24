# typed: strict
# frozen_string_literal: true

class UserEntity < ApplicationEntity
  sig { returns(T::Wikino::DatabaseId) }
  attr_reader :database_id

  sig { returns(String) }
  attr_reader :atname

  sig { returns(String) }
  attr_reader :name

  sig { returns(String) }
  attr_reader :description

  sig do
    params(
      database_id: T::Wikino::DatabaseId,
      atname: String,
      name: String,
      description: String
    ).void
  end
  def initialize(database_id:, atname:, name:, description:)
    @database_id = database_id
    @atname = atname
    @name = name
    @description = description
  end
end
