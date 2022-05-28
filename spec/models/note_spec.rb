# typed: false
# frozen_string_literal: true

RSpec.describe Note, type: :model do
  describe "#titles_in_body" do
    it "returns titles" do
      [
        ["[[a]]", ["a"]],
        ["[[ a ]]", [" a "]],
        ["[[Hello]]", ["Hello"]],
        ["[[こんにちは✌️]]", ["こんにちは✌️"]],
        ["[[a]] [[b]]", ["a", "b"]],
        ["[[Hello]] [[World]]", ["Hello", "World"]],
        ["[[こんにちは]] [[世界🌏]]", ["こんにちは", "世界🌏"]],
        ["[ [a] ]", []],
        ["[[a]", []],
        # A bit weird, but same behavior as Obsidian, Reflect, Bear and etc.
        ["[[[a]]]", ["[a"]],
        ["[[[a]]] [[b]]", ["[a", "b"]],
        ["[[[a]]] [[[b]]]", ["[a", "[b"]],
        ["[[[ a ]]]", ["[ a "]],
      ].each do |(body, expected)|
        note = Note.new(content: NoteContent.new(body:))
        expect(note.titles_in_body).to eq(expected)
      end
    end
  end
end
