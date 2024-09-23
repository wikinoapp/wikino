# typed: strict
# frozen_string_literal: true

class Layouts::BasicLayoutComponent < ApplicationComponent
  T::Sig::WithoutRuntime.sig do
    params(
      joined_lists: List::PrivateRelation,
      main_class_name: String
    ).void
  end
  def initialize(joined_lists:, main_class_name: "")
    @joined_lists = joined_lists
    @main_class_name = main_class_name
  end

  T::Sig::WithoutRuntime.sig { returns(List::PrivateRelation) }
  attr_reader :joined_lists
  private :joined_lists

  sig { returns(String) }
  attr_reader :main_class_name
  private :main_class_name
end
